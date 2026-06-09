// Package engine contains the budget allocation logic that distributes income
// into buckets according to a set of rules. The logic is deliberately free of
// any database concerns so it can be exercised in isolation.
package engine

import "github.com/hjordan6/go_budget/models"

// Allocation records how much a single rule deposited into its bucket.
type Allocation struct {
	RuleID     uint            `json:"rule_id"`
	BucketID   uint            `json:"bucket_id"`
	Type       models.RuleType `json:"type"`
	Amount     float64         `json:"amount"`      // amount deposited by this rule
	NewBalance float64         `json:"new_balance"` // bucket balance after the deposit
}

// Result is the outcome of distributing income across a set of rules.
type Result struct {
	Income      float64      `json:"income"`
	Allocations []Allocation `json:"allocations"`
	Unallocated float64      `json:"unallocated"` // income left after every rule ran
}

// Distribute runs rules against income. Rules are expected to be sorted in
// ascending Priority order; each rule pulls from the income remaining after the
// higher-priority rules have run. The CurrentAmount of each rule's Bucket is
// mutated in place to reflect the deposit, and an Allocation is recorded for
// every rule that deposited a positive amount.
func Distribute(income float64, rules []models.Rule) Result {
	result := Result{Income: income}
	remaining := income

	for i := range rules {
		if remaining <= 0 {
			break
		}

		rule := &rules[i]

		var deposit float64
		switch rule.Type {
		case models.RuleTypeFillUp:
			// Top the bucket up to Target, never beyond, and never more than
			// the income still on hand.
			deposit = min(remaining, max(0, rule.Target-rule.Bucket.CurrentAmount))
		case models.RuleTypePercentage:
			// Take a share of whatever income is left, optionally clamped by a
			// per-execution deposit cap.
			deposit = remaining * rule.Percentage / 100
			if rule.DepositCap != nil {
				deposit = min(deposit, *rule.DepositCap)
			}
			deposit = min(deposit, remaining)
		default:
			// Unknown rule type: deposit nothing.
			continue
		}

		if deposit <= 0 {
			continue
		}

		rule.Bucket.CurrentAmount += deposit
		remaining -= deposit
		result.Allocations = append(result.Allocations, Allocation{
			RuleID:     rule.ID,
			BucketID:   rule.BucketID,
			Type:       rule.Type,
			Amount:     deposit,
			NewBalance: rule.Bucket.CurrentAmount,
		})
	}

	result.Unallocated = remaining
	return result
}
