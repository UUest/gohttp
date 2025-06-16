package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/UUest/gohttp/internal/database"
)

func readiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "Text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func respondWithJSON(w http.ResponseWriter, status int, dat []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
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
	platform       string
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

func (cfg *apiConfig) createChirp(w http.ResponseWriter, r *http.Request) {
	type reqParameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	decoder := json.NewDecoder(r.Body)
	reqParams := reqParameters{}
	err := decoder.Decode(&reqParams)
	if err != nil {
		log.Printf("failed to decode request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	chirpParams := database.CreateChirpParams{}
	valid := true
	if len(reqParams.Body) > 140 {
		chirpParams.Body = "Chirp is too long"
		valid = false
	} else {
		chirpParams.Body = reqParams.Body
	}
	_, rep := chirpCleaner(chirpParams.Body)
	if rep == true {
		chirpParams.Body, _ = chirpCleaner(chirpParams.Body)
	} else {
		chirpParams.Body = reqParams.Body
	}
	chirpParams.ID = uuid.New()
	chirpParams.UserID = reqParams.UserID

	newChirp, err := cfg.dbQueries.CreateChirp(r.Context(), chirpParams)
	type resParameters struct {
		Id         uuid.UUID `json:"id"`
		Body       string    `json:"body"`
		Created_at time.Time `json:"created_at"`
		Updated_at time.Time `json:"updated_at"`
		User_id    uuid.UUID `json:"user_id"`
	}
	resParams := resParameters{
		Id:         newChirp.ID,
		Body:       newChirp.Body,
		Created_at: newChirp.CreatedAt,
		Updated_at: newChirp.UpdatedAt,
		User_id:    newChirp.UserID,
	}
	res, err := json.Marshal(resParams)
	if err != nil {
		log.Printf("failed to marshal response body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if valid == false {
		respondWithError(w, http.StatusBadRequest, res)
	} else {
		respondWithJSON(w, http.StatusCreated, res)
	}
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type reqParameters struct {
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	reqParams := reqParameters{}
	err := decoder.Decode(&reqParams)
	if err != nil {
		log.Printf("failed to decode request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	newUser, err := cfg.dbQueries.CreateUser(r.Context(), reqParams.Email)
	if err != nil {
		log.Printf("failed to create user: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	type resParameters struct {
		Id         uuid.UUID `json:"id"`
		Created_at time.Time `json:"created_at"`
		Updated_at time.Time `json:"updated_at"`
		Email      string    `json:"email"`
	}
	resParams := resParameters{
		Id:         newUser.ID,
		Created_at: newUser.CreatedAt,
		Updated_at: newUser.UpdatedAt,
		Email:      newUser.Email,
	}
	dat, err := json.Marshal(resParams)
	if err != nil {
		log.Printf("failed to marshal response body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	respondWithJSON(w, http.StatusCreated, dat)
}

func (cfg *apiConfig) deleteAllUsers(w http.ResponseWriter, r *http.Request) {
	if cfg.platform == "dev" {
		log.Printf("deleting all users")
		err := cfg.dbQueries.DeleteAllUsers(r.Context())
		if err != nil {
			log.Printf("failed to delete all users: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		respondWithJSON(w, http.StatusOK, nil)
	} else {
		respondWithError(w, http.StatusForbidden, nil)
	}
}
