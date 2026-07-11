package models

import "time"

type Session struct {
	Token     string    `gorm:"primaryKey" json:"token"`
	UserID    uint      `gorm:"index" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
