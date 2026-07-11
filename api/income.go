package api

import (
	"net/http"

	"github.com/hjordan6/go_budget/models"
	"github.com/hjordan6/go_budget/service"
	"gorm.io/gorm"
)

type incomeRequest struct {
	Amount float64 `json:"amount"`
}

type incomeResponse struct {
	Income      models.Income   `json:"income"`
	Allocations []service.Alloc `json:"allocations"`
	Buckets     []models.Bucket `json:"buckets"`
}

// addIncome records an income entry, runs the user's distribution function to
// split it across their buckets, applies the allocations, and returns the
// updated buckets.
func (h *Handler) addIncome(w http.ResponseWriter, r *http.Request) {
	var req incomeRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}
	uid := userID(r)

	var (
		income  models.Income
		allocs  []service.Alloc
		buckets []models.Bucket
	)
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", uid).Order("position asc").Find(&buckets).Error; err != nil {
			return err
		}

		income = models.Income{UserID: uid, Amount: req.Amount}
		if err := tx.Create(&income).Error; err != nil {
			return err
		}

		allocs = service.Distribute(uid, req.Amount, buckets)
		for _, a := range allocs {
			if err := tx.Model(&models.Bucket{}).
				Where("id = ? AND user_id = ?", a.BucketID, uid).
				Update("current_amount", gorm.Expr("current_amount + ?", a.Amount)).
				Error; err != nil {
				return err
			}
		}

		// Reload buckets with the applied amounts.
		return tx.Where("user_id = ?", uid).Order("position asc").Find(&buckets).Error
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not add income")
		return
	}

	writeJSON(w, http.StatusOK, incomeResponse{Income: income, Allocations: allocs, Buckets: buckets})
}

func (h *Handler) listIncomes(w http.ResponseWriter, r *http.Request) {
	var incomes []models.Income
	if err := h.DB.Where("user_id = ?", userID(r)).Order("created_at desc").Limit(50).Find(&incomes).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not load incomes")
		return
	}
	writeJSON(w, http.StatusOK, incomes)
}
