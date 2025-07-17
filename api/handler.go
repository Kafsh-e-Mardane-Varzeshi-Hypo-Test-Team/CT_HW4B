package api

import (
	"log"
	"net/http"
	"time"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	cockroachClient *db.CockroachClient
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

func NewHandler(cockroachClient *db.CockroachClient) *Handler {
	return &Handler{cockroachClient: cockroachClient}
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

	// TODO: Produce to kafka
}
