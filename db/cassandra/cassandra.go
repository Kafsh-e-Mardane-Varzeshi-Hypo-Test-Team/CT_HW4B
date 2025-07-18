package cassandra

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/models"
	"github.com/gocql/gocql"
)

type CassandraClient struct {
	Session *gocql.Session
}

func NewCassandraClient(cfg config.CassandraConfig) (*CassandraClient, error) {
	cluster := gocql.NewCluster(cfg.Host)
	cluster.Port = cfg.Port
	cluster.Keyspace = cfg.Keyspace
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: cfg.User,
		Password: cfg.Password,
	}

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("[cassandra.NewCassandraClient] Failed to create Cassandra session: %v", err)
	}

	log.Println("Successfully connected to Cassandra!")
	return &CassandraClient{Session: session}, nil
}

func (c *CassandraClient) Insert(event models.LogRequest) error {
	query := `INSERT INTO logs.events (
		event_id,
		project_id,
		name,
		time,
		keys
	) VALUES (?, ?, ?, ?, ?)`

	err := c.Session.Query(query, event.EventID,
		event.ProjectID, event.Payload.Name, event.Payload.Timestamp,
		event.Payload.Keys).Exec()
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
			keys
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

	// Set a reasonable fetch size to avoid loading too much data
	iter := c.Session.Query(baseQuery, args...).PageSize(1000).Iter()
	defer iter.Close()

	var events []models.Event
	var eventID gocql.UUID
	var projID gocql.UUID
	var name string
	var timestamp time.Time
	var keys []string

	// Collect all matching events
	for iter.Scan(&eventID, &projID, &name, &timestamp, &keys) {
		// Apply name filter in application if not already filtered in query
		if eventName != "" && name != eventName {
			continue
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
			Keys:      keys,
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
