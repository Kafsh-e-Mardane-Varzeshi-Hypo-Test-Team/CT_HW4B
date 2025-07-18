#!/bin/bash

# Performance Test Suite for Go Project with CockroachDB
BASE_URL="http://localhost:9090/api"
RESULTS_FILE="performance_results_$(date +%Y%m%d_%H%M%S).txt"

# Initialize test data
echo "Initializing test data..."
START_TIME=$(date +%s.%N)

TEST_USER1=$(curl -s -X POST "$BASE_URL/signup" \
  -H "Content-Type: application/json" \
  -d '{"username": "perftestuser1", "password": "password123"}')
USER1_ID=$(echo $TEST_USER1 | jq -r '.user.id')

TEST_PROJECT1=$(curl -s -X POST "$BASE_URL/projects" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER1_ID" \
  -d '{"name": "Perf Test Project 1", "searchableKeys": ["perfkey1", "perfkey2"], "ttl": 3600}')
PROJECT1_ID=$(echo $TEST_PROJECT1 | jq -r '.project.id')
PROJECT1_APIKEY=$(echo $TEST_PROJECT1 | jq -r '.project.apiKey')

END_TIME=$(date +%s.%N)
INIT_TIME=$(echo "$END_TIME - $START_TIME" | bc)
echo "Test data initialized in $INIT_TIME seconds" | tee -a $RESULTS_FILE

# Function to run benchmark and record results
run_benchmark() {
  local test_name=$1
  local iterations=$2
  local command=$3
  
  echo -e "\nRunning $test_name benchmark ($iterations iterations)..." | tee -a $RESULTS_FILE
  START_TIME=$(date +%s.%N)
  
  eval "$command"
  
  END_TIME=$(date +%s.%N)
  ELAPSED=$(echo "$END_TIME - $START_TIME" | bc)
  REQUESTS_PER_SEC=$(echo "scale=2; $iterations / $ELAPSED" | bc)
  
  echo "Results for $test_name:" | tee -a $RESULTS_FILE
  echo "  Total time: $ELAPSED seconds" | tee -a $RESULTS_FILE
  echo "  Requests/sec: $REQUESTS_PER_SEC" | tee -a $RESULTS_FILE
  echo "  Iterations: $iterations" | tee -a $RESULTS_FILE
}

# Benchmark 1: User Signup (Write)
run_benchmark "User Signup" 100 \
  "for i in {1..100}; do
    curl -s -X POST \"$BASE_URL/signup\" \
      -H \"Content-Type: application/json\" \
      -d '{\"username\": \"perfuser\$i\", \"password\": \"password123\"}' > /dev/null &
  done
  wait"

# Benchmark 2: Project Creation (Write)
run_benchmark "Project Creation" 100 \
  "for i in {1..100}; do
    curl -s -X POST \"$BASE_URL/projects\" \
      -H \"Content-Type: application/json\" \
      -H \"X-User-ID: $USER1_ID\" \
      -d '{\"name\": \"Perf Project \$i\", \"searchableKeys\": [\"key\$i\"], \"ttl\": 3600}' > /dev/null &
  done
  wait"

# Benchmark 3: Log Submission (Write)
run_benchmark "Log Submission" 1000 \
  "for i in {1..1000}; do
    curl -s -X POST \"$BASE_URL/logs\" \
      -H \"Content-Type: application/json\" \
      -d '{\"apiKey\": \"$PROJECT1_APIKEY\", \"projectID\": \"$PROJECT1_ID\", \"eventName\": \"perf_event\$i\", \"payload\": {\"key1\": \"value\$i\"}}' > /dev/null &
  done
  wait"

# Sleep to ensure logs are processed
sleep 5

# Benchmark 4: Get All Projects (Read)
run_benchmark "Get All Projects" 200 \
  "for i in {1..200}; do
    curl -s -X GET \"$BASE_URL/projects\" \
      -H \"X-User-ID: $USER1_ID\" > /dev/null &
  done
  wait"

# Benchmark 5: Get Single Project (Read)
run_benchmark "Get Single Project" 200 \
  "for i in {1..200}; do
    curl -s -X GET \"$BASE_URL/projects/$PROJECT1_ID\" \
      -H \"X-User-ID: $USER1_ID\" > /dev/null &
  done
  wait"

# Benchmark 6: Get Project Events (Read)
run_benchmark "Get Project Events" 200 \
  "for i in {1..200}; do
    curl -s -X GET \"$BASE_URL/projects/$PROJECT1_ID/events\" \
      -H \"X-User-ID: $USER1_ID\" > /dev/null &
  done
  wait"

# Benchmark 7: Mixed Read-Write Workload
run_benchmark "Mixed Read-Write Workload" 100 \
  "for i in {1..50}; do
    # Write operation
    curl -s -X POST \"$BASE_URL/logs\" \
      -H \"Content-Type: application/json\" \
      -d '{\"apiKey\": \"$PROJECT1_APIKEY\", \"projectID\": \"$PROJECT1_ID\", \"eventName\": \"mixed_event\$i\", \"payload\": {\"key1\": \"mixed_value\$i\"}}' > /dev/null &
    
    # Read operation
    curl -s -X GET \"$BASE_URL/projects/$PROJECT1_ID/events\" \
      -H \"X-User-ID: $USER1_ID\" > /dev/null &
  done
  wait"

# Benchmark 8: Validate Session (Read)
run_benchmark "Validate Session" 200 \
  "for i in {1..200}; do
    curl -s -X POST \"$BASE_URL/validate-session\" \
      -H \"X-User-ID: $USER1_ID\" > /dev/null &
  done
  wait"

echo -e "\nAll benchmarks completed. Results saved to $RESULTS_FILE"

# Print summary
echo -e "\n=== Performance Test Summary ===" | tee -a $RESULTS_FILE
grep "Results for " $RESULTS_FILE | tee -a $RESULTS_FILE
grep "Requests/sec" $RESULTS_FILE | tee -a $RESULTS_FILE