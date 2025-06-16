package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/UUest/gohttp/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbUrl := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	cfg := &apiConfig{
		dbQueries: database.New(db),
	}
	mux.HandleFunc("GET /api/healthz", readiness)
	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /admin/metrics", cfg.writeMetricsResponse)
	mux.HandleFunc("POST /admin/reset", cfg.resetMetrics)
	mux.HandleFunc("POST /api/validate_chirp", cfg.validateChirp)
	server.ListenAndServe()
	defer server.Shutdown(context.Background())
}
