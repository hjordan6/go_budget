package models

type Rule struct {
	ID          uint                     `gorm:"primaryKey" json:"id"`
	Priority    int                      `gorm:"uniqueIndex;check:priority > 0" json:"priority"`
	Amount      float64                  `json:"amount"`
	Allocations []ProportionalAllocation `gorm:"foreignKey:RuleID" json:"allocations"`
	BucketID    uint                     `json:"bucket_id"`
	Bucket      Bucket                   `gorm:"foreignKey:BucketID" json:"bucket"`
}

type ProportionalAllocation struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	RuleID     uint    `json:"rule_id"`
	Percentage float64 `json:"percentage"`
	Limit      float64 `json:"limit"`
}
