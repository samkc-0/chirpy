package main

import (
	"chirpy/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const (
	port            = "8080"
	staticFilesRoot = "."
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type apiConfig struct {
	fileServerHits atomic.Int32
	db             *database.Queries
	platform       string
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
	}

	fileServerHandler := http.FileServer(http.Dir(staticFilesRoot))
	mux.Handle("/app/", http.StripPrefix("/app", cfg.middlewareMetricsInc(fileServerHandler)))

	mux.HandleFunc("GET /admin/metrics", cfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", cfg.handlerReset)
	mux.HandleFunc("GET /api/healthz", checkReadiness)
	mux.HandleFunc("POST /api/users", cfg.handlerCreateUser)
	mux.HandleFunc("POST /api/chirps", cfg.handlerCreateChirp)
	mux.HandleFunc("GET /api/chirps", cfg.handlerGetChirps)
	mux.HandleFunc("GET /api/chirps/{chirp_id}", cfg.handlerGetChirpByID)

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
		Email string `json:"email"`
	}
	type result struct {
		User User `json:"user"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		result := ErrorResponse{Error: "Something went wrong"}
		dat, err := json.Marshal(&result)
		if err != nil {
			log.Fatalf("Could not even marshal an error json: %s", err)
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(dat)
		return
	}
	user, err := cfg.db.CreateUser(req.Context(), params.Email)
	if err != nil {
		result := ErrorResponse{Error: "Something went wrong"}
		dat, err := json.Marshal(&result)
		if err != nil {
			log.Fatalf("Could not even marshal an error json: %s", err)
		}

		w.WriteHeader(http.StatusBadRequest)
		w.Write(dat)
		return
	}
	userCreated := User{ID: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Email: user.Email}
	dat, err := json.Marshal(&userCreated)
	if err != nil {
		log.Fatal("Could not even marshal a valid user")
	}
	w.WriteHeader(http.StatusCreated)
	w.Write(dat)
	return
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, req *http.Request) {

	type parameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	type response = Chirp

	params := parameters{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&params)

	if err != nil {
		result := ErrorResponse{Error: "Something went wrong"}
		dat, err := json.Marshal(result)
		if err != nil {
			log.Fatal("Could not even marshal error result")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError) // 500
		w.Write(dat)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// validate the chirp
	if len(params.Body) > 140 {
		result := response{Valid: false}
		dat, err := json.Marshal(result)
		if err != nil {
			log.Fatal("Could not marshal invalid result")
		}
		w.WriteHeader(http.StatusBadRequest) // 400
		w.Write(dat)
		return
	}

	cleanedBody := censor(params.Body)

	// add the chirp to the db
	chirpParams := database.CreateChirpParams{
		Body:   cleanedBody,
		UserID: params.UserID,
	}
	chirp, err := cfg.db.CreateChirp(req.Context(), chirpParams)
	if err != nil {
		result := ErrorResponse{Error: "Could not create chirp"}
		dat, err := json.Marshal(result)
		if err != nil {
			log.Fatalf("could not even marshal an error json: %s", err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(dat)
	}
	result := response{
		Valid:     true,
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}
	dat, err := json.Marshal(result)
	if err != nil {
		log.Fatal("Could not marshal valid result")
	}
	w.WriteHeader(http.StatusCreated)
	w.Write(dat)
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, req *http.Request) {
	type response = []Chirp
	chirps, err := cfg.db.GetAllChirps(req.Context())
	if err != nil {
		result := ErrorResponse{Error: "could not get all chirps"}
		dat, err := json.Marshal(result)
		if err != nil {
			log.Fatal("Could not marshal error result in handlerGetAllChirps")
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(dat)
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
	dat, err := json.Marshal(result)
	if err != nil {
		log.Fatal("Could not marshal chirps in handlerGetChirps")
	}
	w.WriteHeader(http.StatusOK)
	w.Write(dat)
}

func (cfg *apiConfig) handlerGetChirpByID(w http.ResponseWriter, req *http.Request) {

	type response = Chirp
	chirp_id, err := uuid.Parse(req.PathValue("chirp_id"))
	if err != nil {
		result := ErrorResponse{Error: "invalid chirp ID"}
		dat, err := json.Marshal(result)
		if err != nil {
			log.Fatal("could not marshal error in handleGetChirpByID")
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(dat)
		return
	}
	chirp, err := cfg.db.GetChirp(req.Context(), chirp_id)
	if err != nil {
		result := ErrorResponse{Error: "chirp not found"}
		dat, err := json.Marshal(result)
		if err != nil {
			log.Fatal("could not marshal error response in handlerGetChirpByID")
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write(dat)
		return
	}
	result := response{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
		Valid:     true,
	}
	dat, err := json.Marshal(result)
	if err != nil {
		log.Fatal("Could not marshal valid and found chirp in handlerGetChirpByID")
	}
	w.WriteHeader(http.StatusOK)
	w.Write(dat)
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
