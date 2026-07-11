package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/hjordan6/go_budget/models"
	"gorm.io/gorm"
)

// Sentinel errors returned from inside the DB transaction closures so the outer
// handler can map them to the right HTTP status.
var (
	errBucketNotFound = errors.New("bucket not found")
	errTxnNotFound    = errors.New("transaction not found")
)

type transactionInput struct {
	BucketID    uint    `json:"bucket_id"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

type transactionResponse struct {
	Transaction models.Transaction `json:"transaction"`
	Buckets     []models.Bucket    `json:"buckets"`
}

// listTransactions returns the current user's transactions, most recent first.
// An optional ?bucket_id= query narrows the list to a single bucket.
func (h *Handler) listTransactions(w http.ResponseWriter, r *http.Request) {
	q := h.DB.Where("user_id = ?", userID(r)).Order("created_at desc").Limit(200)
	if bid := r.URL.Query().Get("bucket_id"); bid != "" {
		id, err := strconv.ParseUint(bid, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid bucket_id")
			return
		}
		q = q.Where("bucket_id = ?", uint(id))
	}
	var txns []models.Transaction
	if err := q.Find(&txns).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not load transactions")
		return
	}
	writeJSON(w, http.StatusOK, txns)
}

// createTransaction records a transaction against one of the user's buckets and
// applies its signed amount to that bucket's balance, atomically.
func (h *Handler) createTransaction(w http.ResponseWriter, r *http.Request) {
	var in transactionInput
	if err := readJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Amount == 0 {
		writeError(w, http.StatusBadRequest, "amount cannot be zero")
		return
	}
	uid := userID(r)

	var txn models.Transaction
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if !ownsBucket(tx, uid, in.BucketID) {
			return errBucketNotFound
		}
		txn = models.Transaction{
			UserID:      uid,
			BucketID:    in.BucketID,
			Amount:      in.Amount,
			Description: in.Description,
		}
		if err := tx.Create(&txn).Error; err != nil {
			return err
		}
		return adjustBucket(tx, uid, in.BucketID, in.Amount)
	})
	if h.handleTxnError(w, err) {
		return
	}
	h.writeTxnResponse(w, http.StatusCreated, uid, txn)
}

// updateTransaction edits a transaction, reversing its old effect and applying
// the new one — including moving it to a different bucket if BucketID changed.
func (h *Handler) updateTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}
	var in transactionInput
	if err := readJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Amount == 0 {
		writeError(w, http.StatusBadRequest, "amount cannot be zero")
		return
	}
	uid := userID(r)

	var txn models.Transaction
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", uint(id), uid).First(&txn).Error; err != nil {
			return errTxnNotFound
		}
		if !ownsBucket(tx, uid, in.BucketID) {
			return errBucketNotFound
		}
		// Reverse the old effect, then apply the new one. When the bucket is
		// unchanged these two updates net out to the delta on one row.
		if err := adjustBucket(tx, uid, txn.BucketID, -txn.Amount); err != nil {
			return err
		}
		if err := adjustBucket(tx, uid, in.BucketID, in.Amount); err != nil {
			return err
		}
		txn.BucketID = in.BucketID
		txn.Amount = in.Amount
		txn.Description = in.Description
		return tx.Save(&txn).Error
	})
	if h.handleTxnError(w, err) {
		return
	}
	h.writeTxnResponse(w, http.StatusOK, uid, txn)
}

// deleteTransaction removes a transaction and reverses its effect on the bucket.
func (h *Handler) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}
	uid := userID(r)

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		var txn models.Transaction
		if err := tx.Where("id = ? AND user_id = ?", uint(id), uid).First(&txn).Error; err != nil {
			return errTxnNotFound
		}
		if err := adjustBucket(tx, uid, txn.BucketID, -txn.Amount); err != nil {
			return err
		}
		return tx.Delete(&txn).Error
	})
	if h.handleTxnError(w, err) {
		return
	}
	var buckets []models.Bucket
	h.DB.Where("user_id = ?", uid).Order("position asc").Find(&buckets)
	writeJSON(w, http.StatusOK, map[string]any{"buckets": buckets})
}

// ownsBucket reports whether the bucket exists and belongs to the user.
func ownsBucket(tx *gorm.DB, uid, bucketID uint) bool {
	var count int64
	tx.Model(&models.Bucket{}).Where("id = ? AND user_id = ?", bucketID, uid).Count(&count)
	return count > 0
}

// adjustBucket adds delta (which may be negative) to a bucket's balance.
func adjustBucket(tx *gorm.DB, uid, bucketID uint, delta float64) error {
	return tx.Model(&models.Bucket{}).Where("id = ? AND user_id = ?", bucketID, uid).
		Update("current_amount", gorm.Expr("current_amount + ?", delta)).Error
}

// handleTxnError writes the appropriate error response for a failed transaction
// closure and reports whether the caller should stop. Returns false on success.
func (h *Handler) handleTxnError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, errTxnNotFound):
		writeError(w, http.StatusNotFound, "transaction not found")
	case errors.Is(err, errBucketNotFound):
		writeError(w, http.StatusNotFound, "bucket not found")
	default:
		writeError(w, http.StatusInternalServerError, "could not save transaction")
	}
	return true
}

// writeTxnResponse returns the saved transaction alongside the user's refreshed
// buckets so the client can update balances in one round trip.
func (h *Handler) writeTxnResponse(w http.ResponseWriter, status int, uid uint, txn models.Transaction) {
	var buckets []models.Bucket
	h.DB.Where("user_id = ?", uid).Order("position asc").Find(&buckets)
	writeJSON(w, status, transactionResponse{Transaction: txn, Buckets: buckets})
}
