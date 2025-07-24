package models

import (
	"time"

	"github.com/gocql/gocql"
)

type LogPayload struct {
	Name      string            `json:"name" binding:"required"`
	Timestamp time.Time         `json:"timestamp" binding:"required"`
	Data      map[string]string `json:"data" binding:"required"`
	TTL       *int64            `json:"ttl,omitempty"`
}

type LogRequest struct {
	EventID   gocql.UUID `json:"event_id"`
	ProjectID string     `json:"project_id" binding:"required"`
	APIKey    string     `json:"api_key" binding:"required"`
	Payload   LogPayload `json:"payload" binding:"required"`
}

// Event represents a log event stored in the database
type Event struct {
	EventID   gocql.UUID        `json:"event_id"`
	ProjectID string            `json:"project_id"`
	Name      string            `json:"name"`
	Timestamp time.Time         `json:"timestamp"`
	Data      map[string]string `json:"data"`
	CreatedAt time.Time         `json:"created_at"`
	TTL       int64             `json:"ttl"`
}

// EventSummary represents aggregated event data for the frontend
type EventSummary struct {
	Name          string    `json:"name"`
	LastTimestamp time.Time `json:"last_timestamp"`
	Count         int       `json:"count"`
}
