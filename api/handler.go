package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/kafka"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	cockroachClient *db.CockroachClient
	kafkaClient     *kafka.KafkaClient
}

type LogPayload struct {
	Name      string    `json:"name" binding:"required"`
	Timestamp time.Time `json:"timestamp" binding:"required"`
	Keys      []string  `json:"keys" binding:"required"`
}

type LogRequest struct {
	ProjectID string     `json:"project_id" binding:"required"`
	APIKey    string     `json:"api_key" binding:"required"`
	Payload   LogPayload `json:"payload" binding:"required"`
}

func NewHandler(cockroachClient *db.CockroachClient, kafkaClient *kafka.KafkaClient) *Handler {
	return &Handler{cockroachClient: cockroachClient, kafkaClient: kafkaClient}
}

func (h *Handler) SubmitLogHandler(c *gin.Context) {
	var req LogRequest

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

	err = h.kafkaClient.ProduceMessage(message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send log to Kafka"})
		log.Printf("[api.SubmitLogHandler] Failed to produce message to Kafka: %v", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Log submitted successfully"})
	log.Printf("[api.SubmitLogHandler] Log submitted successfully for project ID: %s", req.ProjectID)
}
