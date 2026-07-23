package handler

import (
	"bookAPI4/internal/auth"
	"bookAPI4/internal/cache"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	pool   *pgxpool.Pool
	client *redis.Client
}

func NewAuthHandler(pool *pgxpool.Pool, client *redis.Client) *AuthHandler {
	return &AuthHandler{pool: pool, client: client}
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var cred Credentials

	if err := json.NewDecoder(r.Body).Decode(&cred); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if cred.Username == "" || cred.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password required")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cred.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "password not hashed")
		return
	}
	hashstr := string(hash)
	var id int
	err = h.pool.QueryRow(r.Context(), "insert into users (username, password_hash) values ($1, $2) returning id", cred.Username, hashstr).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database issue")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":       id,
		"username": cred.Username,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var cred Credentials

	if err := json.NewDecoder(r.Body).Decode(&cred); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	var id int
	var password string
	err2 := h.pool.QueryRow(r.Context(), "select id, password_hash from users where username=$1", cred.Username).Scan(&id, &password)

	if err2 != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	err3 := bcrypt.CompareHashAndPassword([]byte(password), []byte(cred.Password))
	if err3 != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err4 := auth.GenerateToken(id)
	if err4 != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer") {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Extracting the token out of header
	token := strings.TrimPrefix(header, "Bearer ")

	// Parsing out the exp claim from token
	claims, err := auth.ValidateToken(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	expTime, err := claims.GetExpirationTime()
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ttl := time.Until(expTime.Time)
	if ttl <= 0 {
		writeError(w, http.StatusBadRequest, "token already expired")
		return
	}
	err = cache.BlacklistToken(r.Context(), h.client, token, ttl)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to logout")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "logged out successfully",
	})
}
