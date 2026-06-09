package models

import "time"

// IncomeDistribution records one execution of a user's rules against an income
// amount: how much came in, where it went, and what was left unallocated.
type IncomeDistribution struct {
	ID          uint                     `gorm:"primaryKey" json:"id"`
	UserID      uint                     `json:"user_id"`
	User        User                     `gorm:"foreignKey:UserID" json:"user"`
	Income      float64                  `json:"income"`
	Unallocated float64                  `json:"unallocated"`
	Allocations []DistributionAllocation `gorm:"foreignKey:IncomeDistributionID" json:"allocations"`
	CreatedAt   time.Time                `json:"created_at"`
}

// DistributionAllocation records how much a single rule deposited into its
// bucket during one income distribution.
type DistributionAllocation struct {
	ID                   uint     `gorm:"primaryKey" json:"id"`
	IncomeDistributionID uint     `json:"income_distribution_id"`
	RuleID               uint     `json:"rule_id"`
	BucketID             uint     `json:"bucket_id"`
	Type                 RuleType `json:"type"`
	Amount               float64  `json:"amount"`
	NewBalance           float64  `json:"new_balance"`
}
