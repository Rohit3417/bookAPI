package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Book struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

type BookHandler struct {
	pool *pgxpool.Pool
}

func NewBookHandler(pool *pgxpool.Pool) *BookHandler {
	return &BookHandler{pool: pool}
}

func (h *BookHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), "select id, title, author from books")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var books []Book
	for rows.Next() {
		var b Book
		err = rows.Scan(&b.ID, &b.Title, &b.Author)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "error reading books")
		return
	}
	writeJSON(w, http.StatusOK, books)
}

func (h *BookHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idstr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	var book Book
	row := h.pool.QueryRow(r.Context(), "select id, title, author from books where id=$1", id)
	err = row.Scan(&book.ID, &book.Title, &book.Author)
	if err != nil {
		writeError(w, http.StatusNotFound, "no book found")
		return
	}

	writeJSON(w, http.StatusOK, book)
}

func (h *BookHandler) Create(w http.ResponseWriter, r *http.Request) {
	var b Book
	err := json.NewDecoder(r.Body).Decode(&b)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	err = h.pool.QueryRow(r.Context(),
		"insert into books (title, author) values ($1, $2) returning id",
		b.Title, b.Author,
	).Scan(&b.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create book")
		return
	}

	writeJSON(w, http.StatusCreated, b)
}

func (h *BookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idstr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	commandTag, err := h.pool.Exec(r.Context(), "delete from books where id=$1", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete book")
		return
	}

	if commandTag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Book not found")
		return
	}

	log.Printf("Book with id %v deleted", id)
	w.WriteHeader(http.StatusNoContent)
}

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
