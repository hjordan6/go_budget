package engine

import (
	"testing"

	"github.com/hjordan6/go_budget/models"
)

func capPtr(v float64) *float64 { return &v }

func fillUpRule(priority int, bucketID uint, target, currentAmount float64) models.Rule {
	return models.Rule{
		ID:       uint(priority),
		Priority: priority,
		Type:     models.RuleTypeFillUp,
		BucketID: bucketID,
		Bucket:   models.Bucket{ID: bucketID, CurrentAmount: currentAmount},
		Target:   target,
	}
}

func percentageRule(priority int, bucketID uint, percentage float64, depositCap *float64) models.Rule {
	return models.Rule{
		ID:         uint(priority),
		Priority:   priority,
		Type:       models.RuleTypePercentage,
		BucketID:   bucketID,
		Bucket:     models.Bucket{ID: bucketID},
		Percentage: percentage,
		DepositCap: depositCap,
	}
}

func TestDistribute(t *testing.T) {
	tests := []struct {
		name            string
		income          float64
		rules           []models.Rule
		wantAmounts     []float64 // deposit per emitted allocation, in order
		wantBalances    []float64 // resulting bucket balance per emitted allocation
		wantUnallocated float64
	}{
		{
			name:            "fill_up tops up to target and not beyond",
			income:          1000,
			rules:           []models.Rule{fillUpRule(1, 1, 500, 200)},
			wantAmounts:     []float64{300},
			wantBalances:    []float64{500},
			wantUnallocated: 700,
		},
		{
			name:   "percentage rules see income remaining after earlier rules",
			income: 1000,
			rules: []models.Rule{
				percentageRule(1, 1, 50, nil), // 50% of 1000 = 500
				percentageRule(2, 2, 50, nil), // 50% of remaining 500 = 250
			},
			wantAmounts:     []float64{500, 250},
			wantBalances:    []float64{500, 250},
			wantUnallocated: 250,
		},
		{
			name:            "deposit cap clamps a percentage deposit",
			income:          1000,
			rules:           []models.Rule{percentageRule(1, 1, 50, capPtr(100))}, // 50% = 500, capped to 100
			wantAmounts:     []float64{100},
			wantBalances:    []float64{100},
			wantUnallocated: 900,
		},
		{
			name:            "leftover income is reported as unallocated",
			income:          1000,
			rules:           []models.Rule{percentageRule(1, 1, 25, nil)}, // 250
			wantAmounts:     []float64{250},
			wantBalances:    []float64{250},
			wantUnallocated: 750,
		},
		{
			name:   "income smaller than fill_up target partially fills and later rules get nothing",
			income: 100,
			rules: []models.Rule{
				fillUpRule(1, 1, 500, 0),      // wants 500, only 100 available
				percentageRule(2, 2, 50, nil), // remaining is 0, deposits nothing
			},
			wantAmounts:     []float64{100},
			wantBalances:    []float64{100},
			wantUnallocated: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Distribute(tt.income, tt.rules)

			if got.Income != tt.income {
				t.Errorf("Income = %v, want %v", got.Income, tt.income)
			}
			if got.Unallocated != tt.wantUnallocated {
				t.Errorf("Unallocated = %v, want %v", got.Unallocated, tt.wantUnallocated)
			}
			if len(got.Allocations) != len(tt.wantAmounts) {
				t.Fatalf("got %d allocations, want %d: %+v", len(got.Allocations), len(tt.wantAmounts), got.Allocations)
			}
			for i, a := range got.Allocations {
				if a.Amount != tt.wantAmounts[i] {
					t.Errorf("allocation[%d].Amount = %v, want %v", i, a.Amount, tt.wantAmounts[i])
				}
				if a.NewBalance != tt.wantBalances[i] {
					t.Errorf("allocation[%d].NewBalance = %v, want %v", i, a.NewBalance, tt.wantBalances[i])
				}
			}
		})
	}
}
