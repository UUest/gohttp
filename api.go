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

	"github.com/UUest/gohttp/internal/auth"
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
	jwtSecret      string
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
	token, err := auth.GetBearerToken(r.Header)
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		log.Printf("failed to validate JWTToken: %s", err)
		respondWithError(w, http.StatusUnauthorized, nil)
		return
	}
	type reqParameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	decoder := json.NewDecoder(r.Body)
	reqParams := reqParameters{}
	err = decoder.Decode(&reqParams)
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
	chirpParams.UserID = userID

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
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	reqParams := reqParameters{}
	err := decoder.Decode(&reqParams)
	if err != nil {
		log.Printf("failed to decode request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	hashedPassword, err := auth.HashPassword(reqParams.Password)
	if err != nil {
		log.Printf("failed to hash password: %s", err)
		respondWithError(w, http.StatusBadRequest, []byte("unable to hash password"))
		return
	}
	userParams := database.CreateUserParams{
		Email:          reqParams.Email,
		HashedPassword: hashedPassword,
	}
	newUser, err := cfg.dbQueries.CreateUser(r.Context(), userParams)
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

func (cfg *apiConfig) getChirps(w http.ResponseWriter, r *http.Request) {
	log.Printf("getting all chirps")
	chirps, err := cfg.dbQueries.GetChirps(r.Context())
	if err != nil {
		log.Printf("failed to get all chirps: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	type resParameters struct {
		Id         uuid.UUID `json:"id"`
		Body       string    `json:"body"`
		Created_at time.Time `json:"created_at"`
		Updated_at time.Time `json:"updated_at"`
		User_id    uuid.UUID `json:"user_id"`
	}
	var resParams []resParameters
	for _, chirp := range chirps {
		resParams = append(resParams, resParameters{
			Id:         chirp.ID,
			Body:       chirp.Body,
			Created_at: chirp.CreatedAt,
			Updated_at: chirp.UpdatedAt,
			User_id:    chirp.UserID,
		})
	}
	dat, err := json.Marshal(resParams)
	if err != nil {
		log.Printf("failed to marshal response body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	respondWithJSON(w, http.StatusOK, dat)
}

func (cfg *apiConfig) getChirpByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("chirpID")
	chirpUUID := uuid.MustParse(id)
	if id == "" {
		respondWithError(w, http.StatusBadRequest, nil)
		return
	}
	chirp, err := cfg.dbQueries.GetChirpByID(r.Context(), chirpUUID)
	if err != nil {
		log.Printf("failed to get chirp by id: %s", err)
		respondWithError(w, http.StatusNotFound, nil)
		return
	}
	type resParameters struct {
		Id         uuid.UUID `json:"id"`
		Body       string    `json:"body"`
		Created_at time.Time `json:"created_at"`
		Updated_at time.Time `json:"updated_at"`
		User_id    uuid.UUID `json:"user_id"`
	}
	resParam := resParameters{
		Id:         chirp.ID,
		Body:       chirp.Body,
		Created_at: chirp.CreatedAt,
		Updated_at: chirp.UpdatedAt,
		User_id:    chirp.UserID,
	}
	dat, err := json.Marshal(resParam)
	if err != nil {
		log.Printf("failed to marshal response body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	respondWithJSON(w, http.StatusOK, dat)
}

func (cfg *apiConfig) loginUser(w http.ResponseWriter, r *http.Request) {
	type reqParameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	reqParams := reqParameters{}
	err := decoder.Decode(&reqParams)
	if err != nil {
		log.Printf("failed to decode request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	user, err := cfg.dbQueries.GetUserByEmail(r.Context(), reqParams.Email)
	if err != nil {
		log.Printf("failed to get user by email: %s\n", err)
		respondWithError(w, http.StatusNotFound, []byte("user not found"))
		return
	}
	err = auth.CheckPasswordHash(user.HashedPassword, reqParams.Password)
	if err != nil {
		log.Printf("failed to check password hash: %s", err)
		respondWithError(w, http.StatusUnauthorized, []byte("Incorrect email or password"))
		return
	}
	token, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Duration(3600)*time.Second)
	if err != nil {
		log.Printf("failed to make JWT: %s", err)
		respondWithError(w, http.StatusInternalServerError, []byte("failed to make JWT"))
		return
	}
	if token == "" {
		log.Printf("failed to make JWT: %s", err)
		respondWithError(w, http.StatusInternalServerError, []byte("failed to make JWT"))
		return
	}
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		log.Printf("failed to make refresh token: %s", err)
		respondWithError(w, http.StatusInternalServerError, []byte("failed to make refresh token"))
		return
	}
	rTokenParams := database.CreateRefreshTokenParams{
		UserID: user.ID,
		Token:  refreshToken,
	}
	newRefreshToken, err := cfg.dbQueries.CreateRefreshToken(r.Context(), rTokenParams)
	if err != nil {
		log.Printf("failed to create refresh token: %s", err)
		respondWithError(w, http.StatusInternalServerError, []byte("failed to create refresh token"))
		return
	}
	type resParameters struct {
		Id           uuid.UUID `json:"id"`
		Created_at   time.Time `json:"created_at"`
		Updated_at   time.Time `json:"updated_at"`
		Email        string    `json:"email"`
		Token        string    `json:"token,omitempty"`
		RefreshToken string    `json:"refresh_token,omitempty"`
	}
	resParams := resParameters{
		Id:           user.ID,
		Created_at:   user.CreatedAt,
		Updated_at:   user.UpdatedAt,
		Email:        user.Email,
		Token:        token,
		RefreshToken: newRefreshToken.Token,
	}
	dat, err := json.Marshal(resParams)
	if err != nil {
		log.Printf("failed to marshal response body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	respondWithJSON(w, http.StatusOK, dat)
}

func (cfg *apiConfig) RefreshToken(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("failed to get refresh token: %s", err)
		respondWithError(w, http.StatusUnauthorized, []byte("failed to get refresh token"))
		return
	}
	refreshToken, err := cfg.dbQueries.GetRefreshToken(r.Context(), token)
	if err != nil {
		log.Printf("failed to get refresh token: %s", err)
		respondWithError(w, http.StatusUnauthorized, []byte("failed to get refresh token"))
		return
	}
	if refreshToken.ExpiresAt.Before(time.Now()) {
		log.Printf("refresh token expired")
		respondWithError(w, http.StatusUnauthorized, []byte("refresh token expired"))
		return
	}
	if refreshToken.RevokedAt.Valid {
		log.Printf("refresh token revoked")
		respondWithError(w, http.StatusUnauthorized, []byte("refresh token revoked"))
		return
	}
	user, err := cfg.dbQueries.GetUserByRefreshToken(r.Context(), refreshToken.Token)
	if err != nil {
		log.Printf("failed to get user by refresh token: %s", err)
		respondWithError(w, http.StatusUnauthorized, []byte("failed to get user by refresh token"))
		return
	}
	newToken, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Duration(3600)*time.Second)
	if err != nil {
		log.Printf("failed to make JWT: %s", err)
		respondWithError(w, http.StatusInternalServerError, []byte("failed to make JWT"))
		return
	}
	newRefreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		log.Printf("failed to make refresh token: %s", err)
		respondWithError(w, http.StatusInternalServerError, []byte("failed to make refresh token"))
		return
	}
	rTokenParams := database.CreateRefreshTokenParams{
		UserID: user.ID,
		Token:  newRefreshToken,
	}
	_, err = cfg.dbQueries.CreateRefreshToken(r.Context(), rTokenParams)
	if err != nil {
		log.Printf("failed to create refresh token: %s", err)
		respondWithError(w, http.StatusInternalServerError, []byte("failed to create refresh token"))
		return
	}
	type resParameters struct {
		Token string `json:"token"`
	}
	resParams := resParameters{
		Token: newToken,
	}
	dat, err := json.Marshal(resParams)
	if err != nil {
		log.Printf("failed to marshal response body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	respondWithJSON(w, http.StatusOK, dat)
}

func (cfg *apiConfig) RevokeToken(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("failed to get bearer token: %s", err)
		respondWithError(w, http.StatusUnauthorized, nil)
		return
	}
	err = cfg.dbQueries.RevokeRefreshToken(r.Context(), token)
	if err != nil {
		log.Printf("failed to revoke refresh token: %s", err)
		respondWithError(w, http.StatusInternalServerError, []byte("failed to revoke refresh token"))
		return
	}
	respondWithJSON(w, http.StatusNoContent, nil)
}

func (cfg *apiConfig) updateUser(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		log.Printf("failed to validate JWTToken: %s", err)
		respondWithError(w, http.StatusUnauthorized, nil)
		return
	}
	type reqParameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	reqParams := reqParameters{}
	err = decoder.Decode(&reqParams)
	if err != nil {
		log.Printf("failed to decode request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	hashedPassword, err := auth.HashPassword(reqParams.Password)
	if err != nil {
		log.Printf("failed to hash password: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	updateParams := database.UpdateUserParams{
		ID:             userID,
		Email:          reqParams.Email,
		HashedPassword: hashedPassword,
	}
	updatedUser, err := cfg.dbQueries.UpdateUser(r.Context(), updateParams)
	if err != nil {
		log.Printf("failed to update user: %s", err)
		respondWithError(w, http.StatusInternalServerError, []byte("failed to update user"))
		return
	}
	type resParameters struct {
		Id         uuid.UUID `json:"id"`
		Created_at time.Time `json:"created_at"`
		Updated_at time.Time `json:"updated_at"`
		Email      string    `json:"email"`
	}
	resParams := resParameters{
		Id:         updatedUser.ID,
		Created_at: updatedUser.CreatedAt,
		Updated_at: updatedUser.UpdatedAt,
		Email:      updatedUser.Email,
	}
	dat, err := json.Marshal(resParams)
	if err != nil {
		log.Printf("failed to marshal response body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	respondWithJSON(w, http.StatusOK, dat)
}
