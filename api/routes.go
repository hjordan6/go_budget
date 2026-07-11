package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"gorm.io/gorm"
)

// Handler holds shared dependencies for the HTTP handlers.
type Handler struct {
	DB *gorm.DB
}

// Routes wires up the JSON API under /api and serves the built Vue SPA from
// staticDir for everything else (with client-side routing fallback).
func Routes(db *gorm.DB, staticDir string) *http.ServeMux {
	h := &Handler{DB: db}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	// Auth
	mux.HandleFunc("POST /api/register", h.register)
	mux.HandleFunc("POST /api/login", h.login)
	mux.HandleFunc("POST /api/logout", h.logout)
	mux.HandleFunc("GET /api/me", h.me)

	// Buckets (auth required). The literal /reorder route is more specific than
	// /{id}, so Go's ServeMux dispatches it first.
	mux.HandleFunc("GET /api/buckets", h.requireAuth(h.listBuckets))
	mux.HandleFunc("POST /api/buckets", h.requireAuth(h.createBucket))
	mux.HandleFunc("PUT /api/buckets/reorder", h.requireAuth(h.reorderBuckets))
	mux.HandleFunc("GET /api/buckets/{id}", h.requireAuth(h.getBucket))
	mux.HandleFunc("PUT /api/buckets/{id}", h.requireAuth(h.updateBucket))
	mux.HandleFunc("DELETE /api/buckets/{id}", h.requireAuth(h.deleteBucket))
	mux.HandleFunc("POST /api/buckets/{id}/transfer", h.requireAuth(h.transferMoney))

	// Income (auth required)
	mux.HandleFunc("POST /api/income", h.requireAuth(h.addIncome))
	mux.HandleFunc("GET /api/incomes", h.requireAuth(h.listIncomes))

	// Transactions (auth required)
	mux.HandleFunc("GET /api/transactions", h.requireAuth(h.listTransactions))
	mux.HandleFunc("POST /api/transactions", h.requireAuth(h.createTransaction))
	mux.HandleFunc("PUT /api/transactions/{id}", h.requireAuth(h.updateTransaction))
	mux.HandleFunc("DELETE /api/transactions/{id}", h.requireAuth(h.deleteTransaction))

	// SPA + static assets
	mux.HandleFunc("/", spaHandler(staticDir))

	return mux
}

// spaHandler serves static files from staticDir, falling back to index.html so
// the Vue client-side router can handle unknown paths.
func spaHandler(staticDir string) http.HandlerFunc {
	fileServer := http.FileServer(http.Dir(staticDir))
	index := filepath.Join(staticDir, "index.html")
	return func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean(r.URL.Path)
		p := filepath.Join(staticDir, clean)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, index)
	}
}

// --- small JSON helpers shared across handlers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func readJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
