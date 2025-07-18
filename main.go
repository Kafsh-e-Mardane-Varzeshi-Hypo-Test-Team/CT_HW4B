package main

import (
	"log"
	"net/http"

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

	cockroach, err := cockroach.NewCockroachClient(cfg.CockroachDBConfig)
	if err != nil {
		log.Fatalf("[main] Failed to create CockroachDB client: %v", err)
	}
	err = cockroach.LoadSchema(cfg.CockroachDBConfig)
	if err != nil {
		log.Fatalf("[main] Failed to load db schema: %v", err)
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

	handler := api.NewHandler(cockroach, kafkaProducer, cassandra, clickhouse)
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

	log.Printf("[main] Server starting on port 9090")
	log.Fatalf("[main] Error while running gin router: %v", r.Run(":9090"))
}
