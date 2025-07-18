package models

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID             uuid.UUID `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	APIKey         string    `json:"api_key" db:"api_key"`
	SearchableKeys []string  `json:"searchable_keys" db:"searchable_keys"`
	TTL            *string   `json:"ttl,omitempty" db:"ttl"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type CreateProjectRequest struct {
	Name           string   `json:"name" binding:"required,min=1,max=100"`
	SearchableKeys []string `json:"searchable_keys"`
	TTL            *string  `json:"ttl,omitempty"`
}

type CreateProjectResponse struct {
	Project Project `json:"project"`
	Message string  `json:"message"`
}
