package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/hjordan6/go_budget/models"
	"github.com/hjordan6/go_budget/service"
	"gorm.io/gorm"
)

// Sentinel errors for the staged-transaction workflow, mapped to HTTP statuses
// by handleLMError.
var (
	errLMNotFound     = errors.New("lunch money transaction not found")
	errLMNotPending   = errors.New("transaction is not pending")
	errLMZeroAmount   = errors.New("amount cannot be zero")
	errLMAlreadyFinal = errors.New("transaction already imported")
)

type lmImportResponse struct {
	Transaction models.LunchMoneyTransaction `json:"transaction"`
	Buckets     []models.Bucket              `json:"buckets"`
}

// listLunchMoneyTransactions returns the user's staged Lunch Money rows. Defaults
// to the ones awaiting review; ?status=imported|ignored|all narrows/broadens it.
func (h *Handler) listLunchMoneyTransactions(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = models.LMStatusPending
	}
	q := h.DB.Where("user_id = ?", userID(r)).Order("date desc, id desc").Limit(500)
	if status != "all" {
		q = q.Where("status = ?", status)
	}
	var rows []models.LunchMoneyTransaction
	if err := q.Find(&rows).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not load transactions")
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

type lmUpdateInput struct {
	Payee  *string  `json:"payee"`
	Amount *float64 `json:"amount"`
}

// updateLunchMoneyTransaction edits the payee and/or amount of a pending row so
// the user can tidy it up before importing. Reviewed rows are immutable.
func (h *Handler) updateLunchMoneyTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}
	var in lmUpdateInput
	if err := readJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	var row models.LunchMoneyTransaction
	if err := h.DB.Where("id = ? AND user_id = ?", uint(id), userID(r)).First(&row).Error; err != nil {
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}
	if row.Status != models.LMStatusPending {
		writeError(w, http.StatusBadRequest, "only pending transactions can be edited")
		return
	}
	applyLMEdits(&row, in.Payee, in.Amount)
	if err := h.DB.Save(&row).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not save transaction")
		return
	}
	writeJSON(w, http.StatusOK, row)
}

type lmImportInput struct {
	BucketID uint     `json:"bucket_id"`
	Payee    *string  `json:"payee"` // optional last-minute edits applied atomically
	Amount   *float64 `json:"amount"`
}

// importLunchMoneyTransaction converts a pending row into a real Transaction
// against one of the user's buckets, adjusting that bucket's balance and marking
// the row imported — all in one DB transaction.
func (h *Handler) importLunchMoneyTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}
	var in lmImportInput
	if err := readJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	uid := userID(r)

	var row models.LunchMoneyTransaction
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", uint(id), uid).First(&row).Error; err != nil {
			return errLMNotFound
		}
		if row.Status != models.LMStatusPending {
			return errLMNotPending
		}
		if !ownsBucket(tx, uid, in.BucketID) {
			return errBucketNotFound
		}
		applyLMEdits(&row, in.Payee, in.Amount)
		if row.Amount == 0 {
			return errLMZeroAmount
		}

		txn := models.Transaction{
			UserID:      uid,
			BucketID:    in.BucketID,
			Amount:      row.Amount,
			Description: row.Payee,
		}
		if err := tx.Create(&txn).Error; err != nil {
			return err
		}
		if err := adjustBucket(tx, uid, in.BucketID, row.Amount); err != nil {
			return err
		}
		row.Status = models.LMStatusImported
		row.TransactionID = &txn.ID
		return tx.Save(&row).Error
	})
	if h.handleLMError(w, err) {
		return
	}

	var buckets []models.Bucket
	h.DB.Where("user_id = ?", uid).Order("position asc").Find(&buckets)
	writeJSON(w, http.StatusOK, lmImportResponse{Transaction: row, Buckets: buckets})
}

// ignoreLunchMoneyTransaction marks a pending row as already accounted for so it
// drops out of the review list. Imported rows can't be ignored.
func (h *Handler) ignoreLunchMoneyTransaction(w http.ResponseWriter, r *http.Request) {
	h.setLMStatus(w, r, models.LMStatusIgnored)
}

// restoreLunchMoneyTransaction moves an ignored row back to pending. Imported
// rows can't be restored (they already changed a balance).
func (h *Handler) restoreLunchMoneyTransaction(w http.ResponseWriter, r *http.Request) {
	h.setLMStatus(w, r, models.LMStatusPending)
}

// setLMStatus flips a row between pending and ignored, refusing to touch rows
// that have already been imported.
func (h *Handler) setLMStatus(w http.ResponseWriter, r *http.Request, status string) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}
	var row models.LunchMoneyTransaction
	if err := h.DB.Where("id = ? AND user_id = ?", uint(id), userID(r)).First(&row).Error; err != nil {
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}
	if row.Status == models.LMStatusImported {
		writeError(w, http.StatusBadRequest, "transaction already imported")
		return
	}
	row.Status = status
	if err := h.DB.Save(&row).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not update transaction")
		return
	}
	writeJSON(w, http.StatusOK, row)
}

// syncLunchMoney pulls the current user's recent transactions on demand so they
// don't have to wait for the hourly background sync.
func (h *Handler) syncLunchMoney(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := h.DB.First(&user, userID(r)).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not load account")
		return
	}
	if user.LunchMoneyToken == "" {
		writeError(w, http.StatusBadRequest, "connect your Lunch Money account in Settings first")
		return
	}
	added, err := service.SyncUserLunchMoney(h.DB, user)
	if err != nil {
		writeError(w, http.StatusBadGateway, "lunch money sync failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"added": added})
}

// applyLMEdits applies optional payee/amount overrides to a staged row.
func applyLMEdits(row *models.LunchMoneyTransaction, payee *string, amount *float64) {
	if payee != nil {
		row.Payee = strings.TrimSpace(*payee)
	}
	if amount != nil {
		row.Amount = *amount
	}
}

// handleLMError maps the staging sentinel errors to HTTP responses. Returns
// false (and writes nothing) when err is nil.
func (h *Handler) handleLMError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, errLMNotFound):
		writeError(w, http.StatusNotFound, "transaction not found")
	case errors.Is(err, errBucketNotFound):
		writeError(w, http.StatusNotFound, "bucket not found")
	case errors.Is(err, errLMNotPending), errors.Is(err, errLMAlreadyFinal):
		writeError(w, http.StatusBadRequest, "transaction already reviewed")
	case errors.Is(err, errLMZeroAmount):
		writeError(w, http.StatusBadRequest, "amount cannot be zero")
	default:
		writeError(w, http.StatusInternalServerError, "could not import transaction")
	}
	return true
}
