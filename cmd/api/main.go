package main

import (
	"fmt"
	"net/http"
	"social-network/internal/handlers"
)

func main() {
	http.HandleFunc("/health", handlers.Health)

	fmt.Println("Main server started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
