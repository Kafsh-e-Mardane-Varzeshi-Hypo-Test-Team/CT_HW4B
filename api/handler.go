package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db/cassandra"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db/cockroach"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db/kafka"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/models"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/utils"
	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

type Handler struct {
	cockroachClient *cockroach.CockroachClient
	kafkaProducer   *kafka.Producer
	cassandraClient *cassandra.CassandraClient
}

func NewHandler(cockroachClient *cockroach.CockroachClient, kafkaProducer *kafka.Producer, cassandraClient *cassandra.CassandraClient) *Handler {
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

	req.EventID = gocql.TimeUUID()
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

func (h *Handler) SignupHandler(c *gin.Context) {
	var req models.SignupRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Printf("[api.SignupHandler] Invalid request: %v", err)
		return
	}

	// Check if user already exists
	if h.cockroachClient.UserExists(req.Username) {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		log.Printf("[api.SignupHandler] Username already exists: %s", req.Username)
		return
	}

	// Hash the password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		log.Printf("[api.SignupHandler] Failed to hash password: %v", err)
		return
	}

	// Create the user
	user, err := h.cockroachClient.CreateUser(req.Username, hashedPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		log.Printf("[api.SignupHandler] Failed to create user: %v", err)
		return
	}

	response := models.SignupResponse{
		User:    *user,
		Message: "User created successfully",
	}

	c.JSON(http.StatusCreated, response)
	log.Printf("[api.SignupHandler] User created successfully: %s", user.Username)
}

// LoginHandler handles user authentication
func (h *Handler) LoginHandler(c *gin.Context) {
	var req models.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Printf("[api.LoginHandler] Invalid request: %v", err)
		return
	}

	// Get user by username
	user, err := h.cockroachClient.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		log.Printf("[api.LoginHandler] User not found: %s", req.Username)
		return
	}

	// Check password
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		log.Printf("[api.LoginHandler] Invalid password for user: %s", req.Username)
		return
	}

	response := models.LoginResponse{
		User:    *user,
		Message: "Login successful",
	}

	c.JSON(http.StatusOK, response)
	log.Printf("[api.LoginHandler] User logged in successfully: %s", user.Username)
}

// CreateProjectHandler handles project creation
func (h *Handler) CreateProjectHandler(c *gin.Context) {
	var req models.CreateProjectRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Printf("[api.CreateProjectHandler] Invalid request: %v", err)
		return
	}

	// For now, we'll get the user ID from a header or query parameter
	// In a real application, you'd get this from JWT token or session
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID required"})
		log.Printf("[api.CreateProjectHandler] Missing user ID")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		log.Printf("[api.CreateProjectHandler] Invalid user ID format: %s", userIDStr)
		return
	}

	// Create the project
	project, err := h.cockroachClient.CreateProject(userID, req.Name, req.SearchableKeys, req.TTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		log.Printf("[api.CreateProjectHandler] Failed to create project: %v", err)
		return
	}

	response := models.CreateProjectResponse{
		Project: *project,
		Message: "Project created successfully",
	}

	c.JSON(http.StatusCreated, response)
	log.Printf("[api.CreateProjectHandler] Project created successfully: %s", project.Name)
}

// GetProjectsHandler handles retrieving user's projects
func (h *Handler) GetProjectsHandler(c *gin.Context) {
	// Get user ID from header
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID required"})
		log.Printf("[api.GetProjectsHandler] Missing user ID")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		log.Printf("[api.GetProjectsHandler] Invalid user ID format: %s", userIDStr)
		return
	}

	// Get projects for the user
	projects, err := h.cockroachClient.GetProjectsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve projects"})
		log.Printf("[api.GetProjectsHandler] Failed to get projects: %v", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"projects": projects,
		"count":    len(projects),
	})
	log.Printf("[api.GetProjectsHandler] Retrieved %d projects for user: %s", len(projects), userID)
}
