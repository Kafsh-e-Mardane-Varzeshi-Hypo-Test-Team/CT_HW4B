package cassandra

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/models"
	"github.com/gocql/gocql"
)

type CassandraClient struct {
	Session *gocql.Session
}

func NewCassandraClient(cfg config.CassandraConfig) (*CassandraClient, error) {
	// Create cluster configuration with single host and multiple ports
	var hosts []string
	for _, port := range cfg.Ports {
		hosts = append(hosts, fmt.Sprintf("%s:%d", cfg.Host, port))
	}

	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = cfg.Keyspace

	// Authentication
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: cfg.User,
		Password: cfg.Password,
	}

	// Performance optimizations
	cluster.Consistency = getConsistencyLevel(cfg.Consistency)
	cluster.Timeout = time.Duration(cfg.Timeout) * time.Second
	cluster.ConnectTimeout = time.Duration(cfg.ConnectTimeout) * time.Second

	// Connection pooling
	cluster.NumConns = cfg.NumConns

	// Load balancing and retry policies
	cluster.RetryPolicy = &gocql.ExponentialBackoffRetryPolicy{
		NumRetries: 3,
		Min:        time.Millisecond * 100,
		Max:        time.Second * 2,
	}

	// Use token-aware routing for better performance
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())

	// Enable compression for better network performance
	cluster.Compressor = gocql.SnappyCompressor{}

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("[cassandra.NewCassandraClient] Failed to create Cassandra session: %v", err)
	}

	log.Printf("Successfully connected to Cassandra cluster with hosts: %s", strings.Join(hosts, ", "))
	return &CassandraClient{Session: session}, nil
}

func getConsistencyLevel(consistency string) gocql.Consistency {
	switch strings.ToLower(consistency) {
	case "one":
		return gocql.One
	case "quorum":
		return gocql.Quorum
	case "all":
		return gocql.All
	case "local_quorum":
		return gocql.LocalQuorum
	case "each_quorum":
		return gocql.EachQuorum
	default:
		return gocql.Quorum // Default to quorum for good balance of consistency and performance
	}
}

func (c *CassandraClient) Insert(event models.LogRequest) error {
	// Data is already map[string]string, no conversion needed
	dataMap := event.Payload.Data

	query := `INSERT INTO logs.events (
		event_id,
		project_id,
		name,
		time,
		data
	) VALUES (?, ?, ?, ?, ?)`

	// Use prepared statement for better performance
	stmt := c.Session.Query(query, event.EventID,
		event.ProjectID, event.Payload.Name, event.Payload.Timestamp,
		dataMap)

	// Set consistency level for write operations
	stmt.SetConsistency(gocql.Quorum)

	err := stmt.Exec()
	if err != nil {
		return fmt.Errorf("[cassandra.Insert] Failed to insert event: %v", err)
	}

	log.Printf("[cassandra.Insert] Successfully inserted event: %+v", event)
	return nil
}

// GetEventDetails retrieves detailed event data for a specific event name using Cassandra
func (c *CassandraClient) GetEventDetails(projectID, eventName string, filterKeys []string, offset, limit int) ([]models.Event, error) {
	// Parse projectID to UUID
	projUUID, err := gocql.ParseUUID(projectID)
	if err != nil {
		return nil, fmt.Errorf("[cassandra.GetEventDetails] Invalid project ID: %v", err)
	}

	var args []interface{}
	baseQuery := `
		SELECT 
			event_id,
			project_id,
			name,
			time,
			data
		FROM logs.events 
		WHERE project_id = ?`

	args = append(args, projUUID)

	// Add name filter if provided
	if eventName != "" {
		baseQuery += " AND name = ?"
		args = append(args, eventName)
	}

	// Only add ORDER BY if we're not filtering by name (to avoid secondary index issues)
	if eventName == "" {
		baseQuery += " ORDER BY time DESC"
	}

	// Create query with performance optimizations
	stmt := c.Session.Query(baseQuery, args...)

	// Set consistency level for read operations (can be more relaxed for better performance)
	stmt.SetConsistency(gocql.LocalQuorum)

	// Set page size for efficient pagination
	iter := stmt.PageSize(1000).Iter()
	defer iter.Close()

	var events []models.Event
	var eventID gocql.UUID
	var projID gocql.UUID
	var name string
	var timestamp time.Time
	var dataMap map[string]string

	// Collect all matching events
	for iter.Scan(&eventID, &projID, &name, &timestamp, &dataMap) {
		// Apply name filter in application if not already filtered in query
		if eventName != "" && name != eventName {
			continue
		}

		// Extract keys from data map for filtering
		keys := make([]string, 0, len(dataMap))
		for key := range dataMap {
			keys = append(keys, key)
		}

		// Apply key filters in application (more efficient than ALLOW FILTERING)
		// Use the same logic as ClickHouse: ALL keys must be present (AND logic)
		if len(filterKeys) > 0 {
			allKeysPresent := true
			for _, filterKey := range filterKeys {
				keyFound := false
				for _, eventKey := range keys {
					if eventKey == filterKey {
						keyFound = true
						break
					}
				}
				if !keyFound {
					allKeysPresent = false
					break
				}
			}
			if !allKeysPresent {
				continue
			}
		}

		event := models.Event{
			EventID:   eventID,
			ProjectID: projID.String(),
			Name:      name,
			Timestamp: timestamp,
			Data:      dataMap,
			CreatedAt: timestamp,
		}
		events = append(events, event)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("[cassandra.GetEventDetails] Failed to execute query: %v", err)
	}

	// Sort events by timestamp in descending order if we filtered by name
	if eventName != "" {
		sort.Slice(events, func(i, j int) bool {
			return events[i].Timestamp.After(events[j].Timestamp) // Descending order
		})
	}

	// Apply offset and limit
	if offset >= len(events) {
		return []models.Event{}, nil
	}

	end := offset + limit
	if end > len(events) {
		end = len(events)
	}

	return events[offset:end], nil
}

// Close closes the Cassandra session
func (c *CassandraClient) Close() {
	if c.Session != nil {
		c.Session.Close()
	}
}

// BatchInsert inserts multiple events in a single batch for better performance
func (c *CassandraClient) BatchInsert(events []models.LogRequest) error {
	if len(events) == 0 {
		return nil
	}

	// Create batch for better performance
	batch := c.Session.NewBatch(gocql.LoggedBatch)

	query := `INSERT INTO logs.events (
		event_id,
		project_id,
		name,
		time,
		data
	) VALUES (?, ?, ?, ?, ?)`

	for _, event := range events {
		// Data is already map[string]string, no conversion needed
		dataMap := event.Payload.Data

		batch.Query(query, event.EventID,
			event.ProjectID, event.Payload.Name, event.Payload.Timestamp,
			dataMap)
	}

	// Set consistency level for batch operations
	batch.SetConsistency(gocql.Quorum)

	err := c.Session.ExecuteBatch(batch)
	if err != nil {
		return fmt.Errorf("[cassandra.BatchInsert] Failed to insert batch of %d events: %v", len(events), err)
	}

	log.Printf("[cassandra.BatchInsert] Successfully inserted batch of %d events", len(events))
	return nil
}
