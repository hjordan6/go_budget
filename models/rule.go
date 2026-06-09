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

	// Cap optionally limits how high this rule will take the bucket's
	// balance. A nil Cap means unlimited. Used by percentage rules;
	// fill_up rules are inherently capped by Target.
	Cap *float64 `json:"cap"`
}
