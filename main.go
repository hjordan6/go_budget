package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/hjordan6/go_budget/api"
	"github.com/hjordan6/go_budget/models"
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
		envOr("PGDATABASE", "playground"),
		envOr("PGSSLMODE", "disable"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect: %v", err)
	}

	if err = db.AutoMigrate(
		&models.Bucket{},
		&models.Rule{},
		&models.AutoPayment{},
	); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	addr := ":8888"
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, api.Routes(db)); err != nil {
		log.Fatalf("server: %v", err)
	}
}
