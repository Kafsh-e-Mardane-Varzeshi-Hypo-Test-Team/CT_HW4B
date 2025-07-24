package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/api"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db/cassandra"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db/clickhouse"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db/cockroach"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db/kafka"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// === CockroachDB ===
	cockroachClient, err := connectWithRetry(
		func() (*cockroach.CockroachClient, error) {
			return cockroach.NewCockroachClient(cfg.CockroachDBConfig)
		}, 5, "CockroachDB",
	)
	if err != nil {
		log.Fatalf("[main] %v", err)
	}
	defer cockroachClient.Close()

	// Load schema with retry
	err = retryWithBackoff(func() error {
		return cockroachClient.LoadSchema(cfg.CockroachDBConfig)
	}, 3)
	if err != nil {
		log.Fatalf("[main] Failed to load db schema: %v", err)
	}

	// Log cluster status
	if status, err := cockroachClient.GetClusterStatus(); err == nil {
		log.Printf("[main] CockroachDB cluster: %d nodes, replication factor: %d",
			status["total_nodes"], status["replication_factor"])
	} else {
		log.Printf("[main] Warning: Could not get cluster status: %v", err)
	}

	// === Kafka ===
	err = retryWithBackoff(func() error {
		return kafka.CreateTopic(cfg.KafkaConfig)
	}, 5)
	if err != nil {
		log.Fatalf("[main] Failed to create Kafka topic: %v", err)
	}
	kafkaProducer := kafka.NewProducer(cfg.KafkaConfig)
	defer kafkaProducer.Close()

	// === Cassandra ===
	cassandraClient, err := connectWithRetry(
		func() (*cassandra.CassandraClient, error) {
			return cassandra.NewCassandraClient(cfg.CassandraConfig)
		}, 5, "Cassandra",
	)
	if err != nil {
		log.Fatalf("[main] %v", err)
	}
	defer cassandraClient.Close()

	kafkaConsumerCassandra := kafka.NewConsumer(cfg.KafkaConfig, cassandraClient.Insert, cfg.CassandraConfig.ConsumerGroupId)
	go func() {
		if err := kafkaConsumerCassandra.ConsumeMessages(ctx); err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	// === ClickHouse ===
	clickhouseClient, err := connectWithRetry(
		func() (*clickhouse.ClickHouseClient, error) {
			return clickhouse.NewClickHouseClient(cfg.ClickHouseConfig)
		}, 3, "Clickhouse",
	)
	if err != nil {
		log.Fatalf("[main] %v", err)
	}
	kafkaConsumerClickHouse := kafka.NewConsumer(cfg.KafkaConfig, clickhouseClient.Insert, cfg.ClickHouseConfig.ConsumerGroupId)
	go func() {
		if err := kafkaConsumerClickHouse.ConsumeMessages(ctx); err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	// === Router Setup ===
	handler := api.NewHandler(cockroachClient, kafkaProducer, cassandraClient, clickhouseClient)
	r := api.SetupRouter(handler)

	srv := &http.Server{
		Addr:    ":9090",
		Handler: r,
	}

	go func() {
		log.Printf("[main] Server starting on port 9090")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[main] Server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	<-quit
	log.Println("[main] Shutting down server...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("[main] Server forced to shutdown: %v", err)
	}

	if err := kafkaConsumerCassandra.Close(); err != nil {
		log.Printf("[main] Failed to close Cassandra Kafka consumer: %v", err)
	}

	if err := kafkaConsumerClickHouse.Close(); err != nil {
		log.Printf("[main] Failed to close ClickHouse Kafka consumer: %v", err)
	}

	log.Println("[main] Server Shutdown complete")
}

func connectWithRetry[T any](tryFunc func() (T, error), retries int, name string) (T, error) {
	var client T
	var err error

	for i := 0; i < retries; i++ {
		client, err = tryFunc()
		if err == nil {
			return client, nil
		}
		log.Printf("[%s] Attempt %d: failed to connect: %v", name, i+1, err)
		time.Sleep(time.Duration(1<<uint(i)) * time.Second)
	}
	return client, fmt.Errorf("[%s] failed after %d retries: %w", name, retries, err)
}

func retryWithBackoff(operation func() error, maxRetries int) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := operation()
		if err == nil {
			return nil
		}
		lastErr = err

		// Exponential backoff: 1s, 2s, 4s, 8s, etc.
		backoff := time.Duration(1<<uint(i)) * time.Second
		log.Printf("Operation failed, retrying in %v: %v", backoff, err)
		time.Sleep(backoff)
	}
	return fmt.Errorf("operation failed after %d retries: %v", maxRetries, lastErr)
}
