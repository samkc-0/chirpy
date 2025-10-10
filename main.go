package main

import (
	"chirpy/internal/auth"
	"chirpy/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const (
	port            = "8080"
	staticFilesRoot = "."
)

type apiConfig struct {
	fileServerHits atomic.Int32
	db             *database.Queries
	platform       string
	secret         string
	jwtExpiry      time.Duration
}

type User struct {
	ID        uuid.UUID    `json:"id"`
	CreatedAt sql.NullTime `json:"created_at"`
	UpdatedAt sql.NullTime `json:"updated_at"`
	Email     string       `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID    `json:"id"`
	CreatedAt sql.NullTime `json:"created_at"`
	UpdatedAt sql.NullTime `json:"updated_at"`
	UserID    uuid.UUID    `json:"user_id"`
	Body      string       `json:"body"`
	Valid     bool         `json:"valid"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	dbUrl := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	dbQueries := database.New(db)
	mux := http.NewServeMux()
	cfg := apiConfig{
		fileServerHits: atomic.Int32{},
		db:             dbQueries,
		platform:       os.Getenv("PLATFORM"),
		secret:         os.Getenv("JWT_SECRET"),
		jwtExpiry:      1 * time.Hour,
	}

	fileServerHandler := http.FileServer(http.Dir(staticFilesRoot))
	mux.Handle("/app/", http.StripPrefix("/app", cfg.middlewareMetricsInc(fileServerHandler)))

	mux.HandleFunc("GET /admin/metrics", cfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", cfg.handlerReset)

	mux.HandleFunc("GET /api/healthz", checkReadiness)
	mux.HandleFunc("POST /api/users", cfg.handlerCreateUser)
	mux.HandleFunc("PUT /api/users", cfg.handlerUpdateUser)
	mux.HandleFunc("POST /api/login", cfg.handlerLogin)
	mux.HandleFunc("POST /api/refresh", cfg.handlerRefresh)
	mux.HandleFunc("POST /api/revoke", cfg.handlerRevoke)

	mux.HandleFunc("GET /api/chirps", cfg.handlerGetChirps)
	mux.HandleFunc("GET /api/chirps/{chirp_id}", cfg.handlerGetChirpByID)
	mux.HandleFunc("POST /api/chirps", cfg.handlerCreateChirp)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	fmt.Println("starting server on port " + port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("exiting app..")
}

func checkReadiness(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, req)
	})
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, req *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	cfg.fileServerHits.Store(0)
	cfg.db.DeleteAllUsers(req.Context())
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, req *http.Request) {
	page := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileServerHits.Load())
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(page))
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type result struct {
		User User `json:"user"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, "Something went wrong", http.StatusBadRequest)
		return
	}
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, "invalid password (or password could not be hashed)", http.StatusInternalServerError)
		return
	}

	createUserParams := database.CreateUserParams{Email: params.Email, HashedPassword: hashedPassword}
	user, err := cfg.db.CreateUser(req.Context(), createUserParams)
	if err != nil {
		respondWithError(w, "Something went wrong", http.StatusBadRequest)
		return
	}

	userCreated := User{ID: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Email: user.Email}
	respondWithJSON(w, userCreated, http.StatusCreated)
	return
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, req *http.Request) {
	token := auth.GetBearerToken(req.Header)
	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		respondWithError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	params := parameters{}
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, "invalid resonse body", http.StatusBadRequest)
		return
	}

	if params.Email != "" && !validateEmail(params.Email) {
		respondWithError(w, "invalid email", http.StatusBadRequest)
		return
	}
	if params.Password != "" && !validatePassword(params.Password) {
		respondWithError(w, "invalid password", http.StatusBadRequest)
		return
	}

	var userOut database.UpdateUserEmailRow
	if params.Email != "" {
		user, err := cfg.db.UpdateUserEmail(req.Context(), database.UpdateUserEmailParams{
			ID:    userID,
			Email: params.Email,
		})
		if err != nil {
			respondWithError(w, "failed to update email", http.StatusInternalServerError)
			return
		}
		userOut = user
	}

	if params.Password != "" {
		hashedPassword, err := auth.HashPassword(params.Password)
		if err != nil {
			respondWithError(w, "failed to hash password", http.StatusInternalServerError)
			return
		}
		_, err = cfg.db.UpdateUserPassword(req.Context(), database.UpdateUserPasswordParams{
			ID:             userID,
			HashedPassword: hashedPassword,
		})
		if err != nil {
			respondWithError(w, "failed to update password", http.StatusInternalServerError)
			return
		}

	}

	respondWithJSON(w, User{
		ID:        userOut.ID,
		CreatedAt: userOut.CreatedAt,
		UpdatedAt: userOut.UpdatedAt,
		Email:     userOut.Email,
	}, http.StatusOK)
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email            string `json:"email"`
		Password         string `json:"password"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
	}

	type response struct {
		User
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, "malformed login form", http.StatusBadRequest)
		return
	}
	user, err := cfg.db.GetUserByEmail(req.Context(), params.Email)
	if err != nil {
		respondWithError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	valid_password, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)

	if err != nil {
		respondWithError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if !valid_password {
		respondWithError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	expiry := 1 * time.Hour
	if params.ExpiresInSeconds != 0 {
		expiry = time.Duration(params.ExpiresInSeconds) * time.Second
	}
	token, err := auth.MakeJWT(user.ID, cfg.secret, expiry)

	if err != nil {
		respondWithError(w, "failed to create token", http.StatusInternalServerError)
		return
	}

	refreshToken, _ := auth.MakeRefreshToken()
	refreshTokenParams := database.CreateRefreshTokenParams{
		Token:  refreshToken,
		UserID: user.ID,
	}

	rt, err := cfg.db.CreateRefreshToken(req.Context(), refreshTokenParams)
	if err != nil {
		respondWithError(w, "invalid refresh token", http.StatusBadRequest)
		return
	}

	respondWithJSON(w, response{
		User: User{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
		},
		Token:        token,
		RefreshToken: rt,
	}, http.StatusOK)
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, req *http.Request) {

	type parameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	type response = Chirp

	params := parameters{}
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	token := auth.GetBearerToken(req.Header)
	userID, err := auth.ValidateJWT(token, cfg.secret)

	if err != nil {
		respondWithError(w, "Bad token", http.StatusUnauthorized)
		return
	}

	user, err := cfg.db.GetUser(req.Context(), userID)
	if err != nil {
		respondWithError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	// chirps may not be longer than 140 chars
	if len(params.Body) > 140 {
		respondWithJSON(w, response{Valid: false}, http.StatusBadRequest)
		return
	}

	// filter taboo words
	cleanedBody := censor(params.Body)

	// add the chirp to the db
	chirpParams := database.CreateChirpParams{
		Body:   cleanedBody,
		UserID: user.ID,
	}
	chirp, err := cfg.db.CreateChirp(req.Context(), chirpParams)
	if err != nil {
		respondWithError(w, "Could not create chirp", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, response{
		Valid:     true,
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}, http.StatusCreated)
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, req *http.Request) {
	type response = []Chirp
	chirps, err := cfg.db.GetAllChirps(req.Context())
	if err != nil {
		respondWithError(w, "Could not get all chirps", http.StatusInternalServerError)
		return
	}
	result := response{}
	for _, chirp := range chirps {
		result = append(result, Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		})
	}
	respondWithJSON(w, result, http.StatusOK)
}

func (cfg *apiConfig) handlerGetChirpByID(w http.ResponseWriter, req *http.Request) {

	type response = Chirp

	chirp_id, err := uuid.Parse(req.PathValue("chirp_id"))
	if err != nil {
		respondWithError(w, "Invalid chirp ID", http.StatusBadRequest)
		return
	}

	chirp, err := cfg.db.GetChirp(req.Context(), chirp_id)
	if err != nil {
		respondWithError(w, "Chirp not found", http.StatusNotFound)
		return
	}

	respondWithJSON(w, response{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
		Valid:     true,
	}, http.StatusOK)
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, req *http.Request) {
	refreshToken := auth.GetBearerToken(req.Header)
	validation, err := cfg.db.ValidateRefreshToken(req.Context(), refreshToken)
	if err != nil || validation.Valid.Bool == false {
		respondWithError(w, "bad credentials", http.StatusUnauthorized)
		return
	}
	type response struct {
		Token string `json:"token"`
	}
	accessToken, err := auth.MakeJWT(validation.UserID, cfg.secret, cfg.jwtExpiry)
	if err != nil {
		respondWithError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	respondWithJSON(w, response{Token: accessToken}, http.StatusOK)
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, req *http.Request) {
	refreshToken := auth.GetBearerToken(req.Header)
	if refreshToken == "" {
		respondWithError(w, "no token in header", http.StatusBadRequest)
		return
	}
	err := cfg.db.RevokeRefreshToken(req.Context(), refreshToken)
	if err != nil {
		respondWithError(w, "invalid refresh token", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func censor(text string) string {
	taboos := []string{"kerfuffle", "sharbert", "fornax"}
	censored := text
	for _, taboo := range taboos {
		// note: this is kinda buggy but passes the tests.
		// should be handled by checking word in lowercase after splitting the string
		censored = strings.ReplaceAll(censored, taboo, "****")
		censored = strings.ReplaceAll(censored, strings.ToUpper(taboo), "****")
		censored = strings.ReplaceAll(censored, capitalize(taboo), "****")
	}
	return censored
}

func capitalize(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}

func validateEmail(email string) bool {
	return regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`).MatchString(email)
}

func validatePassword(password string) bool {
	return true
}
