package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync/atomic"

	"github.com/UUest/gohttp/internal/database"
)

func readiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "Text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func respondWithJSON(w http.ResponseWriter, dat []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(dat)
}

func respondWithError(w http.ResponseWriter, status int, dat []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write(dat)
}

func chirpCleaner(chirp string) (string, bool) {
	profaneWords := []string{"kerfuffle", "sharbert", "fornax"}
	replaced := false

	for _, word := range profaneWords {
		pattern := fmt.Sprintf(`(?i)\b%v\b`, regexp.QuoteMeta(word))
		re := regexp.MustCompile(pattern)

		if re.MatchString(chirp) {
			replaced = true
			chirp = re.ReplaceAllString(chirp, "****")
		}
	}

	return chirp, replaced
}

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
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

func (cfg *apiConfig) validateChirp(w http.ResponseWriter, r *http.Request) {
	type reqParameters struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	reqParams := reqParameters{}
	err := decoder.Decode(&reqParams)
	if err != nil {
		log.Printf("failed to decode request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type resParameters struct {
		Error       string `json:"error"`
		Valid       bool   `json:"valid"`
		Body        string `json:"body"`
		CleanedBody string `json:"cleaned_body"`
	}
	resParams := resParameters{}
	if len(reqParams.Body) > 140 {
		resParams.Error = "Chirp is too long"
	} else {
		resParams.Valid = true
		resParams.Body = reqParams.Body
	}
	_, rep := chirpCleaner(resParams.Body)
	if rep == true {
		resParams.CleanedBody, _ = chirpCleaner(resParams.Body)
	} else {
		resParams.Body = reqParams.Body
		resParams.CleanedBody = reqParams.Body
	}
	dat, err := json.Marshal(resParams)
	if err != nil {
		log.Printf("failed to marshal response body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if resParams.Error != "" {
		respondWithError(w, http.StatusBadRequest, dat)
	} else {
		respondWithJSON(w, dat)
	}
}
