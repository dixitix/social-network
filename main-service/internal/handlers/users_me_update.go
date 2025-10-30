package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type updateMeRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	BirthDate *string `json:"birth_date"`
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`
}

func UserMeUpdate(db *sql.DB) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request, userID string) {
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req updateMeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		_, err := db.Exec(
			`update users
			   set first_name = $1,
			       last_name  = $2,
			       birth_date = $3::date,
			       email      = $4,
			       phone      = $5
			 where id = $6`,
			req.FirstName, req.LastName, req.BirthDate, req.Email, req.Phone, userID,
		)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}
