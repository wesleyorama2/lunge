//go:build ignore

// Simple HTTP test server for local performance testing
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	// Simple GET endpoint
	http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"method": r.Method,
			"url":    r.URL.String(),
			"time":   time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	port := "80"
	fmt.Printf("Starting test server on http://localhost:%s\n", port)
	fmt.Println("Endpoints:")
	fmt.Println("  - GET /get")
	fmt.Println("  - GET /health")
	fmt.Println()

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
