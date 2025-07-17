package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/cassandra"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/kafka"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/models"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	cockroachClient *db.CockroachClient
	kafkaProducer   *kafka.Producer
	cassandraClient *cassandra.CassandraClient
}

func NewHandler(cockroachClient *db.CockroachClient, kafkaProducer *kafka.Producer, cassandraClient *cassandra.CassandraClient) *Handler {
	return &Handler{
		cockroachClient: cockroachClient,
		cassandraClient: cassandraClient,
		kafkaProducer:   kafkaProducer,
	}
}

func (h *Handler) SubmitLogHandler(c *gin.Context) {
	var req models.LogRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Printf("[api.SubmitLogHandler] Invalid request: %v", err)
		return
	}

	if !h.cockroachClient.ValidateAPIKey(req.APIKey, req.ProjectID) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key or project ID"})
		log.Printf("[api.SubmitLogHandler] Invalid API key or project ID: %s, %s", req.APIKey, req.ProjectID)
		return
	}

	message, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process log"})
		log.Printf("[api.SubmitLogHandler] Failed to marshal log request: %v", err)
		return
	}

	err = h.kafkaProducer.ProduceMessage(message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send log to Kafka"})
		log.Printf("[api.SubmitLogHandler] Failed to produce message to Kafka: %v", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Log submitted successfully"})
	log.Printf("[api.SubmitLogHandler] Log submitted successfully for project ID: %s", req.ProjectID)
}
