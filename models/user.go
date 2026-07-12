package models

import "time"

type User struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Email        string `gorm:"uniqueIndex" json:"email"`
	Name         string `json:"name"`
	PasswordHash string `json:"-"`
	// LunchMoneyToken is the user's Lunch Money API access token, used by the
	// background sync to pull their transactions. Never serialized to clients.
	LunchMoneyToken string    `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
}
