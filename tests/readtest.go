package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	baseURL        = "http://localhost:9090"
	projectID      = "f3811c65-0075-4fdd-8ac6-600d8596ded9" // Replace with a real project ID
	userID         = "803f80b7-8b49-40e3-9d1f-46e7592b695f" // Replace with a real user ID
	totalRequests  = 5000
	concurrency    = 50
	requestTimeout = 5 * time.Second
)

func main() {
	client := &http.Client{
		Timeout: requestTimeout,
	}

	var wg sync.WaitGroup
	wg.Add(concurrency)

	requestsPerWorker := totalRequests / concurrency

	var mu sync.Mutex
	successCount := 0
	errorCount := 0
	totalDuration := time.Duration(0)

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < requestsPerWorker; j++ {
				url := fmt.Sprintf("%s/api/projects/%s/events", baseURL, projectID)

				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					log.Printf("[ERROR] Building request: %v", err)
					mu.Lock()
					errorCount++
					mu.Unlock()
					continue
				}

				// Set the required user ID header
				req.Header.Set("X-User-ID", userID)

				reqStart := time.Now()
				resp, err := client.Do(req)
				latency := time.Since(reqStart)

				mu.Lock()
				totalDuration += latency
				mu.Unlock()

				if err != nil {
					log.Printf("[ERROR] Request failed: %v", err)
					mu.Lock()
					errorCount++
					mu.Unlock()
					continue
				}

				_, err = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if err != nil || resp.StatusCode != http.StatusOK {
					log.Printf("[ERROR] Response error: %v (status %d)", err, resp.StatusCode)
					mu.Lock()
					errorCount++
					mu.Unlock()
					continue
				}

				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	avgLatency := time.Duration(0)
	if successCount > 0 {
		avgLatency = totalDuration / time.Duration(successCount)
	}

	fmt.Println("====== Read Performance Result ======")
	fmt.Printf("Total requests: %d\n", totalRequests)
	fmt.Printf("Successful: %d, Failed: %d\n", successCount, errorCount)
	fmt.Printf("Total duration: %.2fs\n", elapsed.Seconds())
	fmt.Printf("Average latency: %v\n", avgLatency)
	fmt.Printf("Throughput: %.2f requests/sec\n", float64(successCount)/elapsed.Seconds())
}
