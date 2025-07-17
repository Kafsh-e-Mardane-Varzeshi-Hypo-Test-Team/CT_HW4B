package models

import "time"

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