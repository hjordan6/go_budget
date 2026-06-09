package models

// User owns buckets, rules, and income distributions. Authentication is out of
// scope for now; a user is identified by its ID on each request.
type User struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Name  string `json:"name"`
	Email string `gorm:"uniqueIndex" json:"email"`
}
