package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/hjordan6/go_budget/models"
	"gorm.io/gorm"
)

func Routes(db *gorm.DB) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	mux.HandleFunc("POST /buckets", createBucket(db))
	mux.HandleFunc("POST /rules", createRule(db))
	mux.HandleFunc("POST /auto-payments", createAutoPayment(db))

	return mux
}

// createBucket handles creating a new Bucket from the JSON request body and
// persisting it to the database via GORM.
func createBucket(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var bucket models.Bucket
		if err := json.NewDecoder(r.Body).Decode(&bucket); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Ignore any client-supplied ID; let the database assign it.
		bucket.ID = 0

		if err := db.Create(&bucket).Error; err != nil {
			http.Error(w, "failed to create bucket", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bucket)
	}
}

// createRule handles creating a new Rule from the JSON request body. Any
// allocations included in the body are persisted alongside the rule via
// GORM's has-many association handling.
func createRule(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rule models.Rule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Priority must be a positive integer.
		if rule.Priority <= 0 {
			http.Error(w, "priority must be a positive integer", http.StatusBadRequest)
			return
		}

		// Ignore any client-supplied IDs; let the database assign them.
		rule.ID = 0
		for i := range rule.Allocations {
			rule.Allocations[i].ID = 0
			rule.Allocations[i].RuleID = 0
		}

		// Create cascades to the associated allocations. Omit the Bucket
		// association so only BucketID is used to reference an existing
		// bucket rather than upserting a blank one.
		if err := db.Omit("Bucket").Create(&rule).Error; err != nil {
			// A duplicate priority violates the unique index.
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				http.Error(w, "a rule with this priority already exists", http.StatusConflict)
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
