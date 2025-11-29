package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"
)

func main() {
	// Use all CPU cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Simple handler that responds immediately
	http.HandleFunc("/status/200", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "healthy")
	})

	// Configure server for high throughput
	server := &http.Server{
		Addr:              ":80",
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
		ReadHeaderTimeout: 2 * time.Second,
	}

	log.Printf("Starting high-performance test server on :80")
	log.Printf("Using %d CPU cores", runtime.NumCPU())
	log.Printf("Optimized for load testing - minimal processing overhead")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
