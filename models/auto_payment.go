package models

type AutoPayment struct {
	ID       uint    `gorm:"primaryKey" json:"id"`
	BucketID uint    `json:"bucket_id"`
	Bucket   Bucket  `gorm:"foreignKey:BucketID" json:"bucket"`
	Amount   float64 `json:"amount"`
	Schedule string  `json:"schedule"`
}
