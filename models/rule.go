package models

type Rule struct {
	ID       uint    `gorm:"primaryKey" json:"id"`
	Priority int     `json:"priority"`
	Amount   float64 `json:"amount"`
	BucketID uint    `json:"bucket_id"`
	Bucket   Bucket  `gorm:"foreignKey:BucketID" json:"bucket"`
}
