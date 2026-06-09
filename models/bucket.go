package models

type Bucket struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	MonthlyAmount float64 `json:"monthly_amount"`
	CurrentAmount float64 `json:"current_amount"`
}
