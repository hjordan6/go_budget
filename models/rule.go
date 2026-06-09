package models

// RuleType distinguishes how a rule allocates money from remaining income.
type RuleType string

const (
	// RuleTypeFillUp tops a bucket's balance up to Target, never beyond.
	RuleTypeFillUp RuleType = "fill_up"
	// RuleTypePercentage allocates Percentage of the remaining income.
	RuleTypePercentage RuleType = "percentage"
)

// Rule allocates money into a bucket when income is deposited. Rules execute
// in ascending Priority order, each pulling from the income remaining after
// the higher-priority rules have run.
type Rule struct {
	ID       uint     `gorm:"primaryKey" json:"id"`
	Priority int      `gorm:"uniqueIndex;check:priority > 0" json:"priority"`
	Type     RuleType `gorm:"check:type IN ('fill_up','percentage')" json:"type"`
	BucketID uint     `json:"bucket_id"`
	Bucket   Bucket   `gorm:"foreignKey:BucketID" json:"bucket"`

	// Target is the balance to top the bucket up to. Used by fill_up rules.
	Target float64 `json:"target"`

	// Percentage is the share (0-100) of the remaining income to allocate.
	// Used by percentage rules.
	Percentage float64 `json:"percentage"`

	// DepositCap optionally limits how much this rule deposits in a single
	// execution (the amount taken, not the bucket's total balance). A nil
	// DepositCap means uncapped. Used by percentage rules.
	DepositCap *float64 `json:"deposit_cap"`
}
