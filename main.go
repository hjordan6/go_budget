package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/hjordan6/go_budget/api"
	"github.com/hjordan6/go_budget/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		envOr("PGHOST", "localhost"),
		envOr("PGPORT", "5432"),
		envOr("PGUSER", "postgres"),
		envOr("PGPASSWORD", "jack"),
		envOr("PGDATABASE", "budget"),
		envOr("PGSSLMODE", "disable"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect: %v", err)
	}

	if err = db.AutoMigrate(
		&models.User{},
		&models.Bucket{},
		&models.Income{},
		&models.Session{},
	); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if err = seed(db); err != nil {
		log.Fatalf("seed: %v", err)
	}

	staticDir := envOr("STATIC_DIR", "web/dist")

	addr := ":" + envOr("APP_PORT", "8891")
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, api.Routes(db, staticDir)); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// seed creates a demo user with a handful of buckets so the app is usable
// immediately for testing. It is idempotent — if the demo user already exists
// nothing happens. The demo user's ID drives the sample distribution function
// registered in service/distributions.go.
func seed(db *gorm.DB) error {
	var user models.User
	err := db.Where("email = ?", "demo@example.com").First(&user).Error
	if err == nil {
		return nil // already seeded
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("demo"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user = models.User{Email: "demo@example.com", Name: "Demo", PasswordHash: string(hash)}
	if err := db.Create(&user).Error; err != nil {
		return err
	}

	names := []string{"Food", "Travel", "Entertainment", "Savings"}
	for i, name := range names {
		bucket := models.Bucket{UserID: user.ID, Name: name, Position: i}
		if err := db.Create(&bucket).Error; err != nil {
			return err
		}
	}
	log.Printf("seeded demo user id=%d with %d buckets", user.ID, len(names))
	return nil
}
