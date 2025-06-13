package main

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
)

func readiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "Text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) writeMetricsResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	hitCountString := fmt.Sprintf(
		`<html>
  			<body>
     			<h1>Welcome, Chirpy Admin</h1>
        			<p>Chirpy has been visited %d times!</p>
           	</body>
        </html>`, cfg.fileserverHits.Load())
	w.Write([]byte(hitCountString))
}

func (cfg *apiConfig) resetMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "Text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	cfg.fileserverHits.Store(0)
	w.Write([]byte("Metrics reset"))
}

func main() {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	cfg := &apiConfig{}
	mux.HandleFunc("GET /api/healthz", readiness)
	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /admin/metrics", cfg.writeMetricsResponse)
	mux.HandleFunc("POST /admin/reset", cfg.resetMetrics)
	server.ListenAndServe()
	defer server.Shutdown(context.Background())
}
