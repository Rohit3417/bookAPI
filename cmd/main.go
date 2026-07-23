package main

import (
	"bookAPI4/internal/auth"
	"bookAPI4/internal/cache"
	"bookAPI4/internal/database"
	"bookAPI4/internal/handler"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{
		"error":     msg,
		"code":      status,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")

		if !strings.HasPrefix(header, "Bearer") {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		header = strings.TrimPrefix(header, "Bearer ")
		claims, err := auth.ValidateToken(header)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		mapClaims, ok := claims.(jwt.MapClaims)
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		userID, ok := mapClaims["user_id"].(float64)
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), "userID", int(userID))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("No .env found")
	}

	backgroundctx := context.Background()
	ctx1, cancel1 := context.WithTimeout(backgroundctx, 2*time.Second)
	defer cancel1()
	ctx2, cancel2 := context.WithTimeout(backgroundctx, 2*time.Second)
	defer cancel2()

	pool, err := database.NewPool(ctx1)
	if err != nil {
		log.Fatalf("startup failed : %v", err)
		return
	}
	defer pool.Close()

	client, err := cache.NewRedisClient(ctx2)
	if err != nil {
		log.Fatalf("startup failed : %v", err)
	}
	defer client.Close()

	h := handler.NewBookHandler(pool)
	h2 := handler.NewAuthHandler(pool, client)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
	r.Route("/books", func(r chi.Router) {
		r.Use(AuthMiddleware)
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)
		r.Post("/", h.Create)
		r.Delete("/{id}", h.Delete)
	})

	r.Route("/auth", func(r chi.Router) {
		r.Post("/signup", h2.Signup)
		r.Post("/login", h2.Login)
		r.Post("/logout", h2.Logout)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("server listening on :%v\n", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}

}
