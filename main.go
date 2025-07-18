package main

import (
	"context"
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
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	// Initialize CockroachDB client with retry logic
	var cockroachClient *cockroach.CockroachClient
	var err error

	// Retry connection to CockroachDB cluster
	for i := 0; i < 5; i++ {
		cockroachClient, err = cockroach.NewCockroachClient(cfg.CockroachDBConfig)
		if err == nil {
			break
		}
		log.Printf("[main] Attempt %d: Failed to connect to CockroachDB cluster: %v", i+1, err)
		if i < 4 {
			time.Sleep(time.Duration(1<<uint(i)) * time.Second)
		}
	}

	if err != nil {
		log.Fatalf("[main] Failed to create CockroachDB client after retries: %v", err)
	}
	defer cockroachClient.Close()

	// Load schema with retry
	err = cockroachClient.RetryWithBackoff(func() error {
		return cockroachClient.LoadSchema(cfg.CockroachDBConfig)
	}, 3)
	if err != nil {
		log.Fatalf("[main] Failed to load db schema: %v", err)
	}

	// Log cluster status
	clusterStatus, err := cockroachClient.GetClusterStatus()
	if err != nil {
		log.Printf("[main] Warning: Could not get cluster status: %v", err)
	} else {
		log.Printf("[main] CockroachDB cluster status: %d nodes, replication factor: %d",
			clusterStatus["total_nodes"], clusterStatus["replication_factor"])
	}

	err = kafka.CreateTopic(cfg.KafkaConfig)
	if err != nil {
		log.Fatalf("[main] Failed to create Kafka topic: %v", err)
	}
	kafkaProducer := kafka.NewProducer(cfg.KafkaConfig)

	cassandra, err := cassandra.NewCassandraClient(cfg.CassandraConfig)
	if err != nil {
		log.Fatalf("[main] Failed to create Cassandra client: %v", err)
	}
	kafkaConsumerCassandra := kafka.NewConsumer(cfg.KafkaConfig, cassandra.Insert)
	go kafkaConsumerCassandra.ConsumeMessages()

	clickhouse, err := clickhouse.NewClickHouseClient(cfg.ClickHouseConfig)
	if err != nil {
		log.Fatalf("[main] Failed to create ClickHouse client: %v", err)
	}
	kafkaConsumerClickHouse := kafka.NewConsumer(cfg.KafkaConfig, clickhouse.Insert)
	go kafkaConsumerClickHouse.ConsumeMessages()

	handler := api.NewHandler(cockroachClient, kafkaProducer, cassandra, clickhouse)
	r := gin.Default()

	// Load HTML templates
	r.LoadHTMLGlob("templates/*.html")

	// Serve static files
	r.Static("/static", "./static")

	// Frontend routes
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "home.html", nil)
	})

	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	r.GET("/dashboard", func(c *gin.Context) {
		c.HTML(http.StatusOK, "dashboard.html", nil)
	})

	r.GET("/project/:id", func(c *gin.Context) {
		c.HTML(http.StatusOK, "project.html", nil)
	})

	// API routes
	r.POST("/api/signup", handler.SignupHandler)
	r.POST("/api/login", handler.LoginHandler)
	r.POST("/api/validate-session", handler.ValidateSessionHandler)
	r.POST("/api/projects", handler.CreateProjectHandler)
	r.GET("/api/projects", handler.GetProjectsHandler)
	r.GET("/api/projects/:id", handler.GetProjectHandler)
	r.GET("/api/projects/:id/events", handler.GetEventsHandler)
	r.GET("/api/projects/:id/events/details", handler.GetEventDetailsHandler)
	r.POST("/api/logs", handler.SubmitLogHandler)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		if err := cockroachClient.HealthCheck(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Cluster status endpoint
	r.GET("/cluster-status", func(c *gin.Context) {
		status, err := cockroachClient.GetClusterStatus()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, status)
	})

	// Create server
	srv := &http.Server{
		Addr:    ":9090",
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("[main] Server starting on port 9090")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[main] Error while running gin router: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[main] Shutting down server...")

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("[main] Server forced to shutdown:", err)
	}

	log.Println("[main] Server exited")
}
