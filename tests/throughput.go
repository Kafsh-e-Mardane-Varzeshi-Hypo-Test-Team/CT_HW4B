package main

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"math/rand"
// 	"net/http"
// 	"sync"
// 	"time"
// )

// const (
// 	// totalRequests   = 1000
// 	// concurrency     = 100
// 	requestEndpoint = "http://localhost:9090/api/logs"
// )

// func main() {
// 	rand.Seed(5)

// 	client := &http.Client{
// 		Timeout: 5 * time.Second,
// 	}

// 	var wg sync.WaitGroup
// 	wg.Add(concurrency)

// 	requestsPerWorker := totalRequests / concurrency

// 	start := time.Now()
// 	var mu sync.Mutex
// 	successCount := 0
// 	errorCount := 0

// 	for i := 0; i < concurrency; i++ {
// 		go func(workerID int) {
// 			defer wg.Done()
// 			for j := 0; j < requestsPerWorker; j++ {
// 				reqBody := buildPayload()
// 				bodyBytes, _ := json.Marshal(reqBody)

// 				req, err := http.NewRequest("POST", requestEndpoint, bytes.NewBuffer(bodyBytes))
// 				if err != nil {
// 					log.Printf("[ERROR] Request build: %v", err)
// 					continue
// 				}
// 				req.Header.Set("Content-Type", "application/json")

// 				resp, err := client.Do(req)
// 				if err != nil {
// 					mu.Lock()
// 					errorCount++
// 					mu.Unlock()
// 					continue
// 				}
// 				resp.Body.Close()
// 				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
// 					mu.Lock()
// 					successCount++
// 					mu.Unlock()
// 				} else {
// 					mu.Lock()
// 					errorCount++
// 					mu.Unlock()
// 				}
// 			}
// 		}(i)
// 	}

// 	wg.Wait()
// 	elapsed := time.Since(start)

// 	fmt.Println("====== Performance Result ======")
// 	fmt.Printf("Total requests: %d\n", totalRequests)
// 	fmt.Printf("Successful: %d, Failed: %d\n", successCount, errorCount)
// 	fmt.Printf("Elapsed time: %.2fs\n", elapsed.Seconds())
// 	fmt.Printf("Throughput: %.2f events/sec\n", float64(successCount)/elapsed.Seconds())
// }

// // Payload structure matching your API
// func buildPayload() map[string]interface{} {
// 	userID := randomChoice([]string{"f", "ff", "fff"})
// 	sessionID := randomChoice([]string{"f", "ff", "fff"})

// 	return map[string]interface{}{
// 		"project_id": "f3811c65-0075-4fdd-8ac6-600d8596ded9",
// 		"api_key":    "80b27eed-292e-4993-90d5-69792d762498",
// 		"payload": map[string]interface{}{
// 			"name":      "test_event",
// 			"timestamp": time.Now().Format(time.RFC3339),
// 			"data": map[string]interface{}{
// 				userID:    userID,
// 				sessionID: sessionID,
// 			},
// 		},
// 	}
// }

// func randomChoice(choices []string) string {
// 	return choices[rand.Intn(len(choices))]
// }
