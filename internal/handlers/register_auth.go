package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type registerRequest struct {
	Login string `json:"login"`
	Password string `json:"password"`
}

type registerResponse struct {
	ID int64 `json:"id"`
}

func AuthRegister(db *sql.DB) http.HandlerFunc  {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		req.Login = strings.TrimSpace(req.Login)
		if len(req.Login) < 3 || len(req.Password) < 8 {
			http.Error(w, "validation error", http.StatusBadRequest)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		var id int64
		err = db.QueryRow(
			`insert into users (login, pass_hash) values ($1, $2) returning id`,
			req.Login, 
			hash).Scan(&id)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				http.Error(w, "login already exists", http.StatusConflict)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(registerResponse{ID:id})
	}
}
