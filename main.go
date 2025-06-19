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
	jwtSecret := os.Getenv("JWT_SECRET")
	platform := os.Getenv("PLATFORM")
	polkaKey := os.Getenv("POLKA_KEY")
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
		platform:  platform,
		jwtSecret: jwtSecret,
		polkaKey:  polkaKey,
	}
	mux.HandleFunc("GET /api/healthz", readiness)
	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /admin/metrics", cfg.writeMetricsResponse)
	mux.HandleFunc("POST /admin/reset", cfg.deleteAllUsers)
	mux.HandleFunc("POST /api/users", cfg.createUser)
	mux.HandleFunc("POST /api/chirps", cfg.createChirp)
	mux.HandleFunc("GET /api/chirps", cfg.getChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", cfg.getChirpByID)
	mux.HandleFunc("POST /api/login", cfg.loginUser)
	mux.HandleFunc("POST /api/refresh", cfg.RefreshToken)
	mux.HandleFunc("POST /api/revoke", cfg.RevokeToken)
	mux.HandleFunc("PUT /api/users", cfg.updateUser)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", cfg.deleteChirpByID)
	mux.HandleFunc("POST /api/polka/webhooks", cfg.updateUserChirpyRed)
	server.ListenAndServe()
	defer server.Shutdown(context.Background())
}
