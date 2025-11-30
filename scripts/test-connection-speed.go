//go:build ignore

package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	url := "http://localhost/status/200"
	duration := 10 * time.Second
	concurrency := 100

	fmt.Printf("Testing connection speed to %s\n", url)
	fmt.Printf("Duration: %v, Concurrency: %d\n\n", duration, concurrency)

	// Create optimized HTTP client
	transport := &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		MaxConnsPerHost:     0,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	}
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}

	var (
		totalRequests atomic.Int64
		successCount  atomic.Int64
		errorCount    atomic.Int64
		wg            sync.WaitGroup
	)

	startTime := time.Now()
	endTime := startTime.Add(duration)

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(endTime) {
				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					errorCount.Add(1)
					continue
				}

				resp, err := client.Do(req)
				if err != nil {
					errorCount.Add(1)
					totalRequests.Add(1)
					continue
				}

				// Discard body
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				if resp.StatusCode == 200 {
					successCount.Add(1)
				} else {
					errorCount.Add(1)
				}
				totalRequests.Add(1)
			}
		}()
	}

	// Progress reporter
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		lastCount := int64(0)

		for range ticker.C {
			if time.Now().After(endTime) {
				return
			}
			currentCount := totalRequests.Load()
			rps := currentCount - lastCount
			lastCount = currentCount
			fmt.Printf("Current RPS: %d, Total: %d, Success: %d, Errors: %d\n",
				rps, currentCount, successCount.Load(), errorCount.Load())
		}
	}()

	wg.Wait()
	actualDuration := time.Since(startTime)

	total := totalRequests.Load()
	success := successCount.Load()
	errors := errorCount.Load()
	avgRPS := float64(total) / actualDuration.Seconds()

	fmt.Printf("\n=== Results ===\n")
	fmt.Printf("Total Requests: %d\n", total)
	fmt.Printf("Successful: %d (%.2f%%)\n", success, float64(success)/float64(total)*100)
	fmt.Printf("Errors: %d (%.2f%%)\n", errors, float64(errors)/float64(total)*100)
	fmt.Printf("Duration: %v\n", actualDuration)
	fmt.Printf("Average RPS: %.2f\n", avgRPS)
}
