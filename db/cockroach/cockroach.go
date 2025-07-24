package cockroach

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/models"
)

type CockroachClient struct {
	Db *sql.DB
}

func NewCockroachClient(cfg config.CockroachDBConfig) (*CockroachClient, error) {
	var db *sql.DB
	var err error

	// Try to connect to each port in the cluster
	for i, port := range cfg.Ports {
		connStr := fmt.Sprintf(
			"postgresql://%s@%s:%d/%s?sslmode=disable&connect_timeout=10",
			cfg.User,
			cfg.Host,
			port,
			cfg.Database,
		)

		log.Printf("[db.NewCockroachClient] Attempting to connect to CockroachDB node %d: %s:%d", i+1, cfg.Host, port)

		db, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Printf("[db.NewCockroachClient] Failed to open connection to %s:%d: %v", cfg.Host, port, err)
			continue
		}

		// Set connection pool settings for better performance
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)

		// Test the connection
		err = db.Ping()
		if err != nil {
			log.Printf("[db.NewCockroachClient] Failed to ping %s:%d: %v", cfg.Host, port, err)
			db.Close()
			continue
		}

		log.Printf("[db.NewCockroachClient] Successfully connected to CockroachDB node: %s:%d", cfg.Host, port)
		break
	}

	if db == nil || err != nil {
		// If all nodes failed, try the fallback single host configuration
		log.Printf("[db.NewCockroachClient] All cluster nodes failed, trying fallback connection to %s:%s", cfg.Host, cfg.Port)

		connStr := fmt.Sprintf(
			"postgresql://%s@%s:%s/%s?sslmode=disable",
			cfg.User,
			cfg.Host,
			cfg.Port,
			cfg.Database,
		)

		db, err = sql.Open("postgres", connStr)
		if err != nil {
			return nil, fmt.Errorf("[db.NewCockroachClient] Failed to connect to any CockroachDB node: %v", err)
		}

		err = db.Ping()
		if err != nil {
			return nil, fmt.Errorf("[db.NewCockroachClient] Failed to ping CockroachDB: %v", err)
		}
	}

	log.Println("[db.NewCockroachClient] Successfully connected to CockroachDB!")
	return &CockroachClient{Db: db}, nil
}

func (c *CockroachClient) LoadSchema(cfg config.CockroachDBConfig) error {
	_, err := c.Db.Exec("CREATE DATABASE IF NOT EXISTS " + cfg.Database)
	if err != nil {
		return fmt.Errorf("[db.LoadSchema] Failed to create database: %v", err)
	}

	_, err = c.Db.Exec(`CREATE TABLE IF NOT EXISTS users (
							id UUID PRIMARY KEY,
							username STRING UNIQUE,
							password STRING,
							created_at TIMESTAMP DEFAULT NOW(),
							updated_at TIMESTAMP DEFAULT NOW()
						);

						CREATE TABLE IF NOT EXISTS projects (
							id UUID PRIMARY KEY,
							name STRING,
							user_id UUID REFERENCES users(id),
							api_key STRING UNIQUE,
							searchable_keys STRING[],
							ttl INTERVAL,
							created_at TIMESTAMP DEFAULT NOW(),
							updated_at TIMESTAMP DEFAULT NOW()
						);
						`)
	if err != nil {
		return fmt.Errorf("[db.LoadSchema] Failed to create tables: %v", err)
	}
	log.Println("[db.LoadSchema] Successfully loaded CockroachDB schema!")
	return nil
}

func (c *CockroachClient) ValidateAPIKey(apiKey, projectID string) bool {
	var valid bool
	err := c.Db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM projects WHERE api_key = $1 AND id = $2)",
		apiKey, projectID,
	).Scan(&valid)
	if err != nil {
		log.Printf("API key validation error: %v", err)
	}
	return valid
}

// CreateUser creates a new user in the database
func (c *CockroachClient) CreateUser(username, hashedPassword string) (*models.User, error) {
	user := &models.User{
		ID:        uuid.New(),
		Username:  username,
		Password:  hashedPassword,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := c.Db.Exec(
		"INSERT INTO users (id, username, password, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)",
		user.ID, user.Username, user.Password, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username
func (c *CockroachClient) GetUserByUsername(username string) (*models.User, error) {
	user := &models.User{}
	err := c.Db.QueryRow(
		"SELECT id, username, password, created_at, updated_at FROM users WHERE username = $1",
		username,
	).Scan(&user.ID, &user.Username, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %v", err)
	}
	return user, nil
}

// GetUserByID retrieves a user by ID
func (c *CockroachClient) GetUserByID(userID uuid.UUID) (*models.User, error) {
	user := &models.User{}
	err := c.Db.QueryRow(
		"SELECT id, username, password, created_at, updated_at FROM users WHERE id = $1",
		userID,
	).Scan(&user.ID, &user.Username, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %v", err)
	}
	return user, nil
}

// UserExists checks if a user with the given username exists
func (c *CockroachClient) UserExists(username string) bool {
	var exists bool
	err := c.Db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)",
		username,
	).Scan(&exists)
	if err != nil {
		log.Printf("User existence check error: %v", err)
		return false
	}
	return exists
}

// CreateProject creates a new project for a user
func (c *CockroachClient) CreateProject(userID uuid.UUID, name string, searchableKeys []string, ttl *string) (*models.Project, error) {
	project := &models.Project{
		ID:             uuid.New(),
		Name:           name,
		UserID:         userID,
		APIKey:         uuid.New().String(), // Generate a unique API key
		SearchableKeys: searchableKeys,
		TTL:            ttl,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Convert []string to pq.StringArray for CockroachDB compatibility
	stringArray := pq.StringArray(searchableKeys)

	_, err := c.Db.Exec(
		"INSERT INTO projects (id, name, user_id, api_key, searchable_keys, ttl, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		project.ID, project.Name, project.UserID, project.APIKey, stringArray, project.TTL, project.CreatedAt, project.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %v", err)
	}

	return project, nil
}

// GetProjectsByUserID retrieves all projects for a user
func (c *CockroachClient) GetProjectsByUserID(userID uuid.UUID) ([]models.Project, error) {
	rows, err := c.Db.Query(
		"SELECT id, name, user_id, api_key, searchable_keys, ttl, created_at, updated_at FROM projects WHERE user_id = $1",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects by user ID: %v", err)
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var project models.Project
		var stringArray pq.StringArray
		err := rows.Scan(&project.ID, &project.Name, &project.UserID, &project.APIKey, &stringArray, &project.TTL, &project.CreatedAt, &project.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %v", err)
		}
		// Convert pq.StringArray back to []string
		project.SearchableKeys = []string(stringArray)
		projects = append(projects, project)
	}

	return projects, nil
}

// GetProjectByID retrieves a project by ID
func (c *CockroachClient) GetProjectByID(projectID uuid.UUID) (*models.Project, error) {
	project := &models.Project{}
	var stringArray pq.StringArray

	err := c.Db.QueryRow(
		"SELECT id, name, user_id, api_key, searchable_keys, ttl, created_at, updated_at FROM projects WHERE id = $1",
		projectID,
	).Scan(&project.ID, &project.Name, &project.UserID, &project.APIKey, &stringArray, &project.TTL, &project.CreatedAt, &project.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get project by ID: %v", err)
	}

	// Convert pq.StringArray back to []string
	project.SearchableKeys = []string(stringArray)

	return project, nil
}

// ValidateProjectOwnership checks if a project belongs to a specific user
func (c *CockroachClient) ValidateProjectOwnership(projectID, userID uuid.UUID) bool {
	var exists bool
	err := c.Db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1 AND user_id = $2)",
		projectID, userID,
	).Scan(&exists)
	if err != nil {
		log.Printf("Project ownership validation error: %v", err)
		return false
	}
	return exists
}

// HealthCheck performs a health check on the database connection
func (c *CockroachClient) HealthCheck() error {
	return c.Db.Ping()
}

// GetClusterStatus returns information about the CockroachDB cluster
func (c *CockroachClient) GetClusterStatus() (map[string]interface{}, error) {
	// Query to get cluster information
	rows, err := c.Db.Query(`
		SELECT 
			node_id, 
			address, 
			attrs, 
			locality, 
			started_at,
			updated_at,
			status
		FROM crdb_internal.gossip_nodes 
		ORDER BY node_id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster status: %v", err)
	}
	defer rows.Close()

	var nodes []map[string]interface{}
	for rows.Next() {
		var nodeID, address, attrs, locality, startedAt, updatedAt, status string
		err := rows.Scan(&nodeID, &address, &attrs, &locality, &startedAt, &updatedAt, &status)
		if err != nil {
			log.Printf("Error scanning cluster status row: %v", err)
			continue
		}
		nodes = append(nodes, map[string]interface{}{
			"node_id":    nodeID,
			"address":    address,
			"attrs":      attrs,
			"locality":   locality,
			"started_at": startedAt,
			"updated_at": updatedAt,
			"status":     status,
		})
	}

	// Get replication factor information
	var replicationFactor int
	err = c.Db.QueryRow("SHOW CLUSTER SETTING kv.replication.replication_factor").Scan(&replicationFactor)
	if err != nil {
		log.Printf("Error getting replication factor: %v", err)
		replicationFactor = 3 // Default to 3
	}

	return map[string]interface{}{
		"nodes":              nodes,
		"replication_factor": replicationFactor,
		"total_nodes":        len(nodes),
	}, nil
}

// Close closes the database connection
func (c *CockroachClient) Close() error {
	if c.Db != nil {
		return c.Db.Close()
	}
	return nil
}
