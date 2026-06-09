package api

import (
	"encoding/json"
	"net/http"

	"github.com/hjordan6/go_budget/models"
	"gorm.io/gorm"
)

func Routes(db *gorm.DB) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	mux.HandleFunc("POST /buckets", createBucket(db))

	return mux
}

// createBucket handles creating a new Bucket from the JSON request body and
// persisting it to the database via GORM.
func createBucket(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var bucket models.Bucket
		if err := json.NewDecoder(r.Body).Decode(&bucket); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Ignore any client-supplied ID; let the database assign it.
		bucket.ID = 0

		if err := db.Create(&bucket).Error; err != nil {
			http.Error(w, "failed to create bucket", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bucket)
	}
}
