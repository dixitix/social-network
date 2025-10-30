package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type userProfile struct {
	ID        int64   `json:"id"`
	Login     string  `json:"login"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	BirthDate *string `json:"birth_date"`
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`
}

func UserMe(db *sql.DB) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request, userID string) {
		var u userProfile
		err := db.QueryRow(
			`select id, login, first_name, last_name, birth_date::text, email, phone
			from users where id = $1`, userID,
			).Scan(&u.ID, &u.Login, &u.FirstName, &u.LastName, &u.BirthDate, &u.Email, &u.Phone)
		
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(u)
	})
}
