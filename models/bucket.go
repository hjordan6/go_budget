package models

type Bucket struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	UserID        uint    `json:"user_id"`
	User          User    `gorm:"foreignKey:UserID" json:"user"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	CurrentAmount float64 `json:"current_amount"`
}
