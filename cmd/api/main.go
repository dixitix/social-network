package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"social-network/internal/handlers"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://main_user:main_pass@localhost:5433/main_db?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		panic(fmt.Errorf("db ping failed: %w", err))
	}
	fmt.Println("Connected to Postgres")

	http.HandleFunc("/health", handlers.Health)
	http.HandleFunc("/auth/register", handlers.AuthRegister(db))
	http.HandleFunc("/auth/login", handlers.AuthLogin(db))
	http.HandleFunc("/users/me", handlers.UserMe(db))
	http.HandleFunc("/users/me/update", handlers.UserMeUpdate(db))

	fmt.Println("Main server started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
