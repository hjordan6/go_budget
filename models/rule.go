package models

type Rule struct {
	ID          uint                     `gorm:"primaryKey" json:"id"`
	Priority    int                      `json:"priority"`
	Amount      float64                  `json:"amount"`
	Allocations []ProportionalAllocation `gorm:"embedded" json:"allocation"`
	BucketID    uint                     `json:"bucket_id"`
	Bucket      Bucket                   `gorm:"foreignKey:BucketID" json:"bucket"`
}

type ProportionalAllocation struct {
	Percentage float64 `json:"percentage"`
	Limit      float64 `json:"limit"`
}
