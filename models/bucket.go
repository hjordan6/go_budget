package models

type Bucket struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	UserID        uint    `gorm:"index" json:"user_id"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	CurrentAmount float64 `json:"current_amount"`
	Position      int     `json:"position"`
}
