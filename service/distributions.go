// Package service holds the per-user income distribution functions.
//
// Each user can have a hand-written Go function that decides how a chunk of
// income is split across their buckets. Functions are registered by user ID in
// Registry below. To give a user custom behavior, write a function with the
// DistributeFunc signature and add it to the map. Users without an entry fall
// back to DefaultDistribute (an even split).
//
// A function only touches the buckets it explicitly references, so any bucket
// the user adds later is ignored until the function is updated by hand.
package service

import "github.com/hjordan6/go_budget/models"

// Alloc is a single bucket allocation produced by a distribution function.
type Alloc struct {
	BucketID uint    `json:"bucket_id"`
	Amount   float64 `json:"amount"`
}

// DistributeFunc splits income across the given buckets and returns the amount
// to add to each affected bucket.
type DistributeFunc func(income float64, buckets []models.Bucket) []Alloc

// Registry maps a user ID to their distribution function. The seeded demo user
// (id 1 on a fresh database) gets distributeDemo as a worked example.
var Registry = map[uint]DistributeFunc{
	1: distributeDemo,
	2: distributeJordan,
}

// Distribute runs the user's registered function, or DefaultDistribute if they
// have none yet.
func Distribute(userID uint, income float64, buckets []models.Bucket) []Alloc {
	if fn, ok := Registry[userID]; ok {
		return fn(income, buckets)
	}
	return DefaultDistribute(income, buckets)
}

// distributeDemo is the example function for the demo user. It splits income by
// fixed percentages, matching buckets by name. Buckets whose names are not in
// the weights map (e.g. ones added later) are left untouched.
func distributeDemo(income float64, buckets []models.Bucket) []Alloc {
	weights := map[string]float64{
		"Food":          0.30,
		"Travel":        0.20,
		"Entertainment": 0.15,
		"Savings":       0.35,
	}
	allocs := make([]Alloc, 0, len(buckets))
	for _, b := range buckets {
		if pct, ok := weights[b.Name]; ok && pct > 0 {
			allocs = append(allocs, Alloc{BucketID: b.ID, Amount: income * pct})
		}
	}
	return allocs
}

// distributeJordan implements Jordan's income plan. x is the total income; each
// step is subtracted from the running remainder (xr), and whatever is left over
// lands in "Other". Buckets not named here (e.g. Food, Travel, Car) are left
// untouched. Amounts are matched to buckets by name.
func distributeJordan(income float64, buckets []models.Bucket) []Alloc {
	x := income
	xr := x

	amounts := map[string]float64{}

	tithing := ((x * 0.95) - 227) / 10
	amounts["Tithing"] = tithing
	xr -= tithing

	taxes := x * 0.31
	amounts["Taxes"] = taxes
	xr -= taxes

	amounts["Health Insurance"] = 227
	xr -= 227

	amounts["Rent"] = 413
	xr -= 413

	amounts["Car Insurance"] = 55
	xr -= 55

	amounts["Subscriptions"] = 100
	xr -= 100

	food := xr * 0.185
	if food < 175 {
		food = 175
	} else if food > 400 {
		food = 400
	}
	amounts["Food"] = food
	xr -= food

	if xr > 0 {
		travel := xr * 0.2
		if travel > 375 {
			travel = 375
		}
		amounts["Travel"] = travel
		xr -= travel

		amounts["Car"] = xr * 0.06
		amounts["Clothes"] = xr * 0.045
		amounts["Fun"] = xr * 0.0975
		amounts["Other"] = xr * 0.25
		xr -= (amounts["Other"] + amounts["Car"] + amounts["Clothes"] + amounts["Fun"])
	}

	// Everything remaining goes to Savings.
	amounts["Savings"] = xr

	allocs := make([]Alloc, 0, len(buckets))
	for _, b := range buckets {
		if amt, ok := amounts[b.Name]; ok {
			allocs = append(allocs, Alloc{BucketID: b.ID, Amount: amt})
		}
	}
	return allocs
}

// DefaultDistribute splits income evenly across every bucket. Used for users
// who have not written a custom function yet.
func DefaultDistribute(income float64, buckets []models.Bucket) []Alloc {
	if len(buckets) == 0 {
		return nil
	}
	share := income / float64(len(buckets))
	allocs := make([]Alloc, 0, len(buckets))
	for _, b := range buckets {
		allocs = append(allocs, Alloc{BucketID: b.ID, Amount: share})
	}
	return allocs
}
