package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db/cassandra"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db/clickhouse"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db/cockroach"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db/kafka"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/models"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/utils"
	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

type Handler struct {
	cockroachClient  *cockroach.CockroachClient
	kafkaProducer    *kafka.Producer
	cassandraClient  *cassandra.CassandraClient
	clickhouseClient *clickhouse.ClickHouseClient
}

func NewHandler(cockroachClient *cockroach.CockroachClient, kafkaProducer *kafka.Producer, cassandraClient *cassandra.CassandraClient, clickhouseClient *clickhouse.ClickHouseClient) *Handler {
	return &Handler{
		cockroachClient:  cockroachClient,
		cassandraClient:  cassandraClient,
		clickhouseClient: clickhouseClient,
		kafkaProducer:    kafkaProducer,
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

	projectTTL, err := h.cockroachClient.GetProjectTTL(req.ProjectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve project TTL"})
		log.Printf("[api.SubmitLogHandler] Failed to get project TTL: %v", err)
		return
	}
	if projectTTL != nil {
		req.Payload.TTL = projectTTL
	}

	req.EventID = gocql.TimeUUID()
	message, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process log"})
		log.Printf("[api.SubmitLogHandler] Failed to marshal log request: %v", err)
		return
	}

	err = h.kafkaProducer.ProduceMessage(c, message)
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

// GetProjectHandler handles retrieving a specific project
func (h *Handler) GetProjectHandler(c *gin.Context) {
	projectIDStr := c.Param("id")
	userIDStr := c.GetHeader("X-User-ID")

	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID format"})
		return
	}

	// Validate project ownership
	if !h.cockroachClient.ValidateProjectOwnership(projectID, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get project details
	project, err := h.cockroachClient.GetProjectByID(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve project"})
		return
	}

	c.JSON(http.StatusOK, project)
}

// GetEventsHandler handles retrieving events for a project
func (h *Handler) GetEventsHandler(c *gin.Context) {
	projectIDStr := c.Param("id")
	userIDStr := c.GetHeader("X-User-ID")

	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID format"})
		return
	}

	// Validate project ownership
	if !h.cockroachClient.ValidateProjectOwnership(projectID, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get filter keys from query parameters
	filterKeys := []string{}
	if keysParam := c.Query("keys"); keysParam != "" {
		filterKeys = strings.Split(keysParam, ",")
	}

	// Get event summaries from ClickHouse (much faster for aggregations)
	summaries, err := h.clickhouseClient.GetEventSummaries(projectID.String(), filterKeys)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve events"})
		log.Printf("[api.GetEventsHandler] Failed to get events: %v", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": summaries,
		"total":  len(summaries),
	})
}

// GetEventDetailsHandler handles retrieving detailed event information
func (h *Handler) GetEventDetailsHandler(c *gin.Context) {
	projectIDStr := c.Param("id")
	userIDStr := c.GetHeader("X-User-ID")

	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID format"})
		return
	}

	if !h.cockroachClient.ValidateProjectOwnership(projectID, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	eventName := c.Query("name")
	if eventName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Event name required"})
		return
	}

	filterKeys := []string{}
	if keysParam := c.Query("keys"); keysParam != "" {
		filterKeys = strings.Split(keysParam, ",")
	}

	offset := 0
	if offsetParam := c.Query("offset"); offsetParam != "" {
		if offsetVal, err := strconv.Atoi(offsetParam); err == nil {
			offset = offsetVal
		}
	}

	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if limitVal, err := strconv.Atoi(limitParam); err == nil {
			limit = limitVal
		}
	}

	// Get total from ClickHouse (more efficient than Cassandra)
	total, err := h.clickhouseClient.GetEventCount(projectID.String(), eventName, filterKeys)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve total count"})
		log.Printf("[api.GetEventDetailsHandler] Failed to get event count: %v", err)
		return
	}

	print(total)

	events, err := h.cassandraClient.GetEventDetails(projectID.String(), eventName, filterKeys, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve event details"})
		log.Printf("[api.GetEventDetailsHandler] Failed to get event details: %v", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  total,
	})
}

// ValidateSessionHandler validates if a user session is still valid
func (h *Handler) ValidateSessionHandler(c *gin.Context) {
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Check if user exists in database
	user, err := h.cockroachClient.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session"})
		log.Printf("[api.ValidateSessionHandler] User not found: %s", userID)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"user":  user,
	})
	log.Printf("[api.ValidateSessionHandler] Session validated for user: %s", user.Username)
}

// OptimizeClickHouseTableHandler triggers TTL deletion and table optimization
func (h *Handler) OptimizeClickHouseTableHandler(c *gin.Context) {
	err := h.clickhouseClient.OptimizeTable()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to optimize ClickHouse table"})
		log.Printf("[api.OptimizeClickHouseTableHandler] Failed to optimize table: %v", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ClickHouse table optimization completed successfully",
	})
	log.Printf("[api.OptimizeClickHouseTableHandler] ClickHouse table optimization completed")
}

// GetClickHouseTTLStatusHandler returns TTL processing status
func (h *Handler) GetClickHouseTTLStatusHandler(c *gin.Context) {
	status, err := h.clickhouseClient.GetTTLStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get TTL status"})
		log.Printf("[api.GetClickHouseTTLStatusHandler] Failed to get TTL status: %v", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ttl_status": status,
		"count":      len(status),
	})
	log.Printf("[api.GetClickHouseTTLStatusHandler] Retrieved TTL status for %d partitions", len(status))
}
