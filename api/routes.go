package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/hjordan6/go_budget/engine"
	"github.com/hjordan6/go_budget/models"
	"gorm.io/gorm"
)

func Routes(db *gorm.DB) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	mux.HandleFunc("POST /users", createUser(db))
	mux.HandleFunc("POST /buckets", createBucket(db))
	mux.HandleFunc("POST /rules", createRule(db))
	mux.HandleFunc("POST /auto-payments", createAutoPayment(db))
	mux.HandleFunc("POST /income", incomeDistribution(db))

	return mux
}

// createUser handles creating a new User from the JSON request body and
// persisting it to the database via GORM.
func createUser(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user models.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Ignore any client-supplied ID; let the database assign it.
		user.ID = 0

		if err := db.Create(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				http.Error(w, "a user with this email already exists", http.StatusConflict)
				return
			}
			http.Error(w, "failed to create user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
	}
}

// createBucket handles creating a new Bucket from the JSON request body and
// persisting it to the database via GORM. Each bucket belongs to a user.
func createBucket(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var bucket models.Bucket
		if err := json.NewDecoder(r.Body).Decode(&bucket); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if bucket.UserID == 0 {
			http.Error(w, "user_id is required", http.StatusBadRequest)
			return
		}

		// Ignore any client-supplied ID; let the database assign it.
		bucket.ID = 0

		// Omit the User association so only UserID is used to reference an
		// existing user rather than upserting a blank one.
		if err := db.Omit("User").Create(&bucket).Error; err != nil {
			if errors.Is(err, gorm.ErrForeignKeyViolated) {
				http.Error(w, "user does not exist", http.StatusBadRequest)
				return
			}
			http.Error(w, "failed to create bucket", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bucket)
	}
}

// createRule handles creating a new Rule from the JSON request body. Rules
// are validated according to their type before being persisted via GORM.
func createRule(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rule models.Rule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if rule.UserID == 0 {
			http.Error(w, "user_id is required", http.StatusBadRequest)
			return
		}

		// Priority must be a positive integer.
		if rule.Priority <= 0 {
			http.Error(w, "priority must be a positive integer", http.StatusBadRequest)
			return
		}

		// Validate fields according to the rule type.
		switch rule.Type {
		case models.RuleTypeFillUp:
			if rule.Target <= 0 {
				http.Error(w, "fill_up rules require a positive target", http.StatusBadRequest)
				return
			}
		case models.RuleTypePercentage:
			if rule.Percentage <= 0 || rule.Percentage > 100 {
				http.Error(w, "percentage rules require a percentage between 0 and 100", http.StatusBadRequest)
				return
			}
			if rule.DepositCap != nil && *rule.DepositCap < 0 {
				http.Error(w, "deposit_cap must not be negative", http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "type must be 'fill_up' or 'percentage'", http.StatusBadRequest)
			return
		}

		// Ignore any client-supplied ID; let the database assign it.
		rule.ID = 0

		// Omit the User and Bucket associations so only the foreign keys are
		// used to reference existing rows rather than upserting blank ones.
		if err := db.Omit("User", "Bucket").Create(&rule).Error; err != nil {
			// A duplicate priority for this user violates the unique index.
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				http.Error(w, "a rule with this priority already exists for this user", http.StatusConflict)
				return
			}
			if errors.Is(err, gorm.ErrForeignKeyViolated) {
				http.Error(w, "user or bucket does not exist", http.StatusBadRequest)
				return
			}
			http.Error(w, "failed to create rule", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(rule)
	}
}

// incomeDistribution accepts an income amount for a user and distributes it
// into that user's buckets by executing their rules in ascending priority
// order. The rules are run via engine.Distribute; the resulting bucket balances
// and a record of the distribution are persisted inside a transaction so the
// deposit is applied atomically.
func incomeDistribution(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			UserID uint    `json:"user_id"`
			Amount float64 `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.UserID == 0 {
			http.Error(w, "user_id is required", http.StatusBadRequest)
			return
		}
		if req.Amount <= 0 {
			http.Error(w, "amount must be a positive number", http.StatusBadRequest)
			return
		}

		distribution := models.IncomeDistribution{UserID: req.UserID, Income: req.Amount}
		err := db.Transaction(func(tx *gorm.DB) error {
			// Make sure the user exists before recording anything against them.
			var count int64
			if err := tx.Model(&models.User{}).Where("id = ?", req.UserID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return errUserNotFound
			}

			// Only this user's rules run, into this user's buckets.
			var rules []models.Rule
			if err := tx.Preload("Bucket").
				Where("user_id = ?", req.UserID).
				Order("priority asc").
				Find(&rules).Error; err != nil {
				return err
			}

			result := engine.Distribute(req.Amount, rules)
			distribution.Unallocated = result.Unallocated

			for _, a := range result.Allocations {
				// Persist the bucket's new balance.
				if err := tx.Model(&models.Bucket{}).
					Where("id = ?", a.BucketID).
					Update("current_amount", a.NewBalance).Error; err != nil {
					return err
				}
				distribution.Allocations = append(distribution.Allocations, models.DistributionAllocation{
					RuleID:     a.RuleID,
					BucketID:   a.BucketID,
					Type:       a.Type,
					Amount:     a.Amount,
					NewBalance: a.NewBalance,
				})
			}

			// Save the distribution and its allocations as a history record.
			return tx.Omit("User").Create(&distribution).Error
		})
		if err != nil {
			if errors.Is(err, errUserNotFound) {
				http.Error(w, "user does not exist", http.StatusBadRequest)
				return
			}
			http.Error(w, "failed to distribute income", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(distribution)
	}
}

// errUserNotFound signals that a request referenced a user that does not exist.
var errUserNotFound = errors.New("user not found")

// createAutoPayment handles creating a new AutoPayment from the JSON request
// body and persisting it to the database via GORM.
func createAutoPayment(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var autoPayment models.AutoPayment
		if err := json.NewDecoder(r.Body).Decode(&autoPayment); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Ignore any client-supplied ID; let the database assign it.
		autoPayment.ID = 0

		// Omit the Bucket association so only BucketID is used to reference
		// an existing bucket rather than upserting a blank one.
		if err := db.Omit("Bucket").Create(&autoPayment).Error; err != nil {
			http.Error(w, "failed to create auto payment", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(autoPayment)
	}
}
