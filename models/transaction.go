package models

import "time"

// Transaction records a single change to a bucket's balance: an expense
// (negative Amount) or a deposit/adjustment (positive Amount). Creating a
// transaction adjusts the linked bucket's CurrentAmount by Amount; editing or
// deleting one reverses and re-applies that effect so balances stay in sync.
type Transaction struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index" json:"user_id"`
	BucketID    uint      `gorm:"index" json:"bucket_id"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
