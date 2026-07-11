package service

import (
	"math"
	"testing"

	"github.com/hjordan6/go_budget/models"
)

const eps = 1e-9

// bucketsNamed builds a bucket slice, assigning each a 1-based ID in order so
// allocations can be matched back by BucketID in assertions.
func bucketsNamed(names ...string) []models.Bucket {
	bs := make([]models.Bucket, len(names))
	for i, n := range names {
		bs[i] = models.Bucket{ID: uint(i + 1), Name: n}
	}
	return bs
}

// allocByID indexes allocations by bucket ID for convenient lookup.
func allocByID(allocs []Alloc) map[uint]float64 {
	m := make(map[uint]float64, len(allocs))
	for _, a := range allocs {
		m[a.BucketID] = a.Amount
	}
	return m
}

// total sums every allocation amount.
func total(allocs []Alloc) float64 {
	var sum float64
	for _, a := range allocs {
		sum += a.Amount
	}
	return sum
}

func TestDefaultDistribute(t *testing.T) {
	t.Run("empty buckets returns nil", func(t *testing.T) {
		if got := DefaultDistribute(100, nil); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("splits evenly", func(t *testing.T) {
		bs := bucketsNamed("A", "B", "C", "D")
		allocs := DefaultDistribute(100, bs)
		if len(allocs) != len(bs) {
			t.Fatalf("expected %d allocations, got %d", len(bs), len(allocs))
		}
		for _, a := range allocs {
			if math.Abs(a.Amount-25) > eps {
				t.Errorf("bucket %d: expected 25, got %v", a.BucketID, a.Amount)
			}
		}
		if math.Abs(total(allocs)-100) > eps {
			t.Errorf("total = %v, want 100", total(allocs))
		}
	})
}

func TestDistributeDemo(t *testing.T) {
	t.Run("splits by named percentages and conserves income", func(t *testing.T) {
		bs := bucketsNamed("Food", "Travel", "Entertainment", "Savings")
		allocs := distributeDemo(1000, bs)
		by := allocByID(allocs)

		want := map[uint]float64{
			1: 300, // Food 30%
			2: 200, // Travel 20%
			3: 150, // Entertainment 15%
			4: 350, // Savings 35%
		}
		if len(allocs) != len(want) {
			t.Fatalf("expected %d allocations, got %d", len(want), len(allocs))
		}
		for id, w := range want {
			if math.Abs(by[id]-w) > eps {
				t.Errorf("bucket %d: expected %v, got %v", id, w, by[id])
			}
		}
		if math.Abs(total(allocs)-1000) > eps {
			t.Errorf("total = %v, want 1000 (weights should sum to 1.0)", total(allocs))
		}
	})

	t.Run("ignores buckets not in the weights map", func(t *testing.T) {
		bs := bucketsNamed("Food", "Groceries", "Vacation")
		allocs := distributeDemo(500, bs)
		if len(allocs) != 1 {
			t.Fatalf("expected only Food to be allocated, got %d allocations", len(allocs))
		}
		if allocs[0].BucketID != 1 || math.Abs(allocs[0].Amount-150) > eps {
			t.Errorf("expected Food (id 1) = 150, got id %d = %v", allocs[0].BucketID, allocs[0].Amount)
		}
	})

	t.Run("no matching buckets returns empty, not nil", func(t *testing.T) {
		allocs := distributeDemo(500, bucketsNamed("Rent"))
		if allocs == nil {
			t.Fatal("expected non-nil empty slice")
		}
		if len(allocs) != 0 {
			t.Fatalf("expected 0 allocations, got %d", len(allocs))
		}
	})
}

func TestDistributeJordan(t *testing.T) {
	// Every named bucket in the plan. Because each step is subtracted from the
	// running remainder and Savings absorbs whatever is left, the allocations
	// must sum back to the original income when all buckets are present.
	allNames := []string{
		"Tithing", "Taxes", "Health Insurance", "Rent", "Car Insurance",
		"Subscriptions", "Food", "Travel", "Car", "Clothes", "Fun", "Other", "Savings",
	}

	t.Run("conserves income across all buckets", func(t *testing.T) {
		for _, income := range []float64{2500, 5000, 12000} {
			bs := bucketsNamed(allNames...)
			allocs := distributeJordan(income, bs)
			if math.Abs(total(allocs)-income) > 1e-6 {
				t.Errorf("income %v: total = %v, want %v", income, total(allocs), income)
			}
		}
	})

	t.Run("fixed-amount buckets get their fixed values", func(t *testing.T) {
		bs := bucketsNamed("Health Insurance", "Rent", "Car Insurance", "Subscriptions")
		by := allocByID(distributeJordan(5000, bs))
		fixed := map[uint]float64{1: 227, 2: 413, 3: 55, 4: 100}
		for id, w := range fixed {
			if math.Abs(by[id]-w) > eps {
				t.Errorf("bucket %d: expected fixed %v, got %v", id, w, by[id])
			}
		}
	})

	t.Run("food is clamped to its floor on low income", func(t *testing.T) {
		// On low income the 18.5%-of-remainder food figure falls under the 175
		// floor and should be pinned there.
		by := allocByID(distributeJordan(2000, bucketsNamed("Food")))
		if math.Abs(by[1]-175) > eps {
			t.Errorf("expected Food clamped to 175, got %v", by[1])
		}
	})

	t.Run("food is clamped to its ceiling on high income", func(t *testing.T) {
		by := allocByID(distributeJordan(50000, bucketsNamed("Food")))
		if math.Abs(by[1]-400) > eps {
			t.Errorf("expected Food clamped to 400, got %v", by[1])
		}
	})

	t.Run("ignores buckets not named in the plan", func(t *testing.T) {
		allocs := distributeJordan(5000, bucketsNamed("Groceries", "Vacation"))
		if len(allocs) != 0 {
			t.Fatalf("expected 0 allocations for unknown buckets, got %d", len(allocs))
		}
	})
}

func TestDistributeDispatch(t *testing.T) {
	t.Run("registered user uses their function", func(t *testing.T) {
		// User 1 is the demo user: Food should get 30%.
		bs := bucketsNamed("Food", "Travel", "Entertainment", "Savings")
		by := allocByID(Distribute(1, 1000, bs))
		if math.Abs(by[1]-300) > eps {
			t.Errorf("user 1 (demo): expected Food = 300, got %v", by[1])
		}
	})

	t.Run("unregistered user falls back to even split", func(t *testing.T) {
		bs := bucketsNamed("A", "B")
		allocs := Distribute(999, 100, bs)
		if len(allocs) != 2 {
			t.Fatalf("expected 2 allocations, got %d", len(allocs))
		}
		for _, a := range allocs {
			if math.Abs(a.Amount-50) > eps {
				t.Errorf("bucket %d: expected even split 50, got %v", a.BucketID, a.Amount)
			}
		}
	})

	t.Run("unregistered user with no buckets returns nil", func(t *testing.T) {
		if got := Distribute(999, 100, nil); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})
}
