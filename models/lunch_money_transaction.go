package models

import "time"

// LunchMoneyTransaction is a transaction pulled from the Lunch Money API into a
// staging area for review. It is NOT the same as models.Transaction: nothing
// here touches a bucket balance until the user imports it. Each row starts as
// "pending" and moves to "imported" (converted into a real Transaction against a
// bucket) or "ignored" (the user already accounted for it manually).
//
// Amount is stored in this app's signed convention (negative = expense), already
// converted from Lunch Money's opposite convention at import time. Payee and
// Amount are editable while the row is pending so the user can tidy them up
// before importing.
type LunchMoneyTransaction struct {
	ID           uint    `gorm:"primaryKey" json:"id"`
	UserID       uint    `gorm:"uniqueIndex:idx_lm_user_txn" json:"user_id"`
	LunchMoneyID int64   `gorm:"uniqueIndex:idx_lm_user_txn" json:"lunch_money_id"`
	Date         string  `json:"date"` // YYYY-MM-DD, as reported by Lunch Money
	Payee        string  `json:"payee"`
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	Notes        string  `json:"notes"`
	// Status is one of "pending", "imported", "ignored".
	Status string `gorm:"index" json:"status"`
	// TransactionID links to the real Transaction created on import, if any.
	TransactionID *uint     `json:"transaction_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// LunchMoneyTransaction status values.
const (
	LMStatusPending  = "pending"
	LMStatusImported = "imported"
	LMStatusIgnored  = "ignored"
)
