package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/hjordan6/go_budget/models"
	"gorm.io/gorm"
)

func (h *Handler) listBuckets(w http.ResponseWriter, r *http.Request) {
	var buckets []models.Bucket
	if err := h.DB.Where("user_id = ?", userID(r)).Order("position asc").Find(&buckets).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not load buckets")
		return
	}
	writeJSON(w, http.StatusOK, buckets)
}

// findOwnedBucket loads a bucket by id, ensuring it belongs to the current user.
func (h *Handler) findOwnedBucket(w http.ResponseWriter, r *http.Request) (*models.Bucket, bool) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid bucket id")
		return nil, false
	}
	var b models.Bucket
	err = h.DB.Where("id = ? AND user_id = ?", uint(id), userID(r)).First(&b).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(w, http.StatusNotFound, "bucket not found")
		return nil, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return nil, false
	}
	return &b, true
}

func (h *Handler) getBucket(w http.ResponseWriter, r *http.Request) {
	b, ok := h.findOwnedBucket(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, b)
}

type bucketInput struct {
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	CurrentAmount float64 `json:"current_amount"`
}

func (h *Handler) createBucket(w http.ResponseWriter, r *http.Request) {
	var in bucketInput
	if err := readJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	uid := userID(r)

	// Append at the end: new buckets are ignored by the distribution function
	// until it is manually updated to reference them.
	var maxPos int
	h.DB.Model(&models.Bucket{}).Where("user_id = ?", uid).
		Select("COALESCE(MAX(position), -1)").Scan(&maxPos)

	b := models.Bucket{
		UserID:        uid,
		Name:          in.Name,
		Description:   in.Description,
		CurrentAmount: in.CurrentAmount,
		Position:      maxPos + 1,
	}
	if err := h.DB.Create(&b).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not create bucket")
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (h *Handler) updateBucket(w http.ResponseWriter, r *http.Request) {
	b, ok := h.findOwnedBucket(w, r)
	if !ok {
		return
	}
	var in bucketInput
	if err := readJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	b.Name = in.Name
	b.Description = in.Description
	b.CurrentAmount = in.CurrentAmount
	if err := h.DB.Save(b).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not update bucket")
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (h *Handler) deleteBucket(w http.ResponseWriter, r *http.Request) {
	b, ok := h.findOwnedBucket(w, r)
	if !ok {
		return
	}
	if err := h.DB.Delete(b).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete bucket")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

type transferRequest struct {
	ToBucketID uint    `json:"to_bucket_id"`
	Amount     float64 `json:"amount"`
}

// transferMoney moves an amount from the bucket in the path to another bucket
// owned by the same user. Both buckets are updated in one transaction.
func (h *Handler) transferMoney(w http.ResponseWriter, r *http.Request) {
	from, ok := h.findOwnedBucket(w, r)
	if !ok {
		return
	}
	var req transferRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}
	if req.ToBucketID == from.ID {
		writeError(w, http.StatusBadRequest, "cannot transfer to the same bucket")
		return
	}
	if req.Amount > from.CurrentAmount {
		writeError(w, http.StatusBadRequest, "not enough in this bucket to move that much")
		return
	}
	uid := userID(r)

	var to models.Bucket
	if err := h.DB.Where("id = ? AND user_id = ?", req.ToBucketID, uid).First(&to).Error; err != nil {
		writeError(w, http.StatusNotFound, "destination bucket not found")
		return
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Bucket{}).Where("id = ? AND user_id = ?", from.ID, uid).
			Update("current_amount", gorm.Expr("current_amount - ?", req.Amount)).Error; err != nil {
			return err
		}
		return tx.Model(&models.Bucket{}).Where("id = ? AND user_id = ?", to.ID, uid).
			Update("current_amount", gorm.Expr("current_amount + ?", req.Amount)).Error
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not move money")
		return
	}

	var buckets []models.Bucket
	h.DB.Where("user_id = ?", uid).Order("position asc").Find(&buckets)
	writeJSON(w, http.StatusOK, buckets)
}

type reorderRequest struct {
	IDs []uint `json:"ids"`
}

// reorderBuckets rewrites the position of the user's buckets to match the given
// order. Only buckets owned by the user are affected.
func (h *Handler) reorderBuckets(w http.ResponseWriter, r *http.Request) {
	var req reorderRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	uid := userID(r)
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		for pos, id := range req.IDs {
			if err := tx.Model(&models.Bucket{}).
				Where("id = ? AND user_id = ?", id, uid).
				Update("position", pos).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not reorder buckets")
		return
	}

	var buckets []models.Bucket
	h.DB.Where("user_id = ?", uid).Order("position asc").Find(&buckets)
	writeJSON(w, http.StatusOK, buckets)
}
