package cassandra

import (
	"fmt"
	"log"
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

	// Build query with optional TTL
	var query string
	if event.Payload.TTL != nil && *event.Payload.TTL > 0 {
		query = `INSERT INTO logs.events (
			event_id,
			project_id,
			name,
			time,
			data
		) VALUES (?, ?, ?, ?, ?) USING TTL ?`

		// Use prepared statement for better performance
		stmt := c.Session.Query(query, event.EventID,
			event.ProjectID, event.Payload.Name, event.Payload.Timestamp,
			dataMap, int(*event.Payload.TTL))

		// Set consistency level for write operations
		stmt.SetConsistency(gocql.Quorum)

		err := stmt.Exec()
		if err != nil {
			return fmt.Errorf("[cassandra.Insert] Failed to insert event: %v", err)
		}

		log.Printf("[cassandra.Insert] Successfully inserted event with TTL of %d seconds: %s", *event.Payload.TTL, event.Payload.Name)
		return nil
	} else {
		query = `INSERT INTO logs.events (
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

		log.Printf("[cassandra.Insert] Successfully inserted event: %s", event.Payload.Name)
		return nil
	}
}

// GetEventDetails retrieves detailed event data for a specific event name using Cassandra
func (c *CassandraClient) GetEventDetails(projectID, eventName string, filterKeys []string, offset, limit int) ([]models.Event, error) {
	projUUID, err := gocql.ParseUUID(projectID)
	if err != nil {
		return nil, fmt.Errorf("[cassandra.GetEventDetails] Invalid project ID: %v", err)
	}

	var args []interface{}
	query := `
		SELECT 
			event_id,
			project_id,
			name,
			time,
			data
		FROM logs.events 
		WHERE project_id = ? AND name = ?
	`

	args = append(args, projUUID, eventName)

	stmt := c.Session.Query(query, args...).PageSize(1000).Consistency(gocql.LocalQuorum)

	iter := stmt.Iter()
	defer iter.Close()

	var (
		events    []models.Event
		eventID   gocql.UUID
		projID    gocql.UUID
		name      string
		timestamp time.Time
		dataMap   map[string]string
		count     int
	)

	for iter.Scan(&eventID, &projID, &name, &timestamp, &dataMap) {
		// filter keys (manually like before)
		keys := make([]string, 0, len(dataMap))
		for k := range dataMap {
			keys = append(keys, k)
		}

		matches := true
		for _, filter := range filterKeys {
			found := false
			for _, k := range keys {
				if k == filter {
					found = true
					break
				}
			}
			if !found {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}

		if count < offset {
			count++
			continue
		}

		if len(events) >= limit {
			break
		}

		events = append(events, models.Event{
			EventID:   eventID,
			ProjectID: projID.String(),
			Name:      name,
			Timestamp: timestamp,
			Data:      dataMap,
			CreatedAt: timestamp,
		})
		count++
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("[cassandra.GetEventDetails] Iteration error: %v", err)
	}

	return events, nil
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

	// Separate queries for events with and without TTL
	queryWithoutTTL := `INSERT INTO logs.events (
		event_id,
		project_id,
		name,
		time,
		data
	) VALUES (?, ?, ?, ?, ?)`

	queryWithTTL := `INSERT INTO logs.events (
		event_id,
		project_id,
		name,
		time,
		data
	) VALUES (?, ?, ?, ?, ?) USING TTL ?`

	ttlCount := 0
	noTtlCount := 0

	for _, event := range events {
		// Data is already map[string]string, no conversion needed
		dataMap := event.Payload.Data

		if event.Payload.TTL != nil && *event.Payload.TTL > 0 {
			batch.Query(queryWithTTL, event.EventID,
				event.ProjectID, event.Payload.Name, event.Payload.Timestamp,
				dataMap, int(*event.Payload.TTL))
			ttlCount++
		} else {
			batch.Query(queryWithoutTTL, event.EventID,
				event.ProjectID, event.Payload.Name, event.Payload.Timestamp,
				dataMap)
			noTtlCount++
		}
	}

	// Set consistency level for batch operations
	batch.SetConsistency(gocql.Quorum)

	err := c.Session.ExecuteBatch(batch)
	if err != nil {
		return fmt.Errorf("[cassandra.BatchInsert] Failed to insert batch of %d events: %v", len(events), err)
	}

	log.Printf("[cassandra.BatchInsert] Successfully inserted batch of %d events (%d with TTL, %d without TTL)", len(events), ttlCount, noTtlCount)
	return nil
}
