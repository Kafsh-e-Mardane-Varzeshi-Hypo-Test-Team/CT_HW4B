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

// GetEventSummaries retrieves aggregated event data for a project
func (c *CassandraClient) GetEventSummaries(projectID string, filterKeys []string) ([]models.EventSummary, error) {
	var query string
	var args []interface{}

	if len(filterKeys) > 0 {
		// Build query with key filters
		keyConditions := make([]string, len(filterKeys))
		for i, key := range filterKeys {
			keyConditions[i] = "keys CONTAINS ?"
			args = append(args, key)
		}
		query = fmt.Sprintf(`SELECT name, time FROM logs.events 
			WHERE project_id = ? AND %s 
			ORDER BY time DESC ALLOW FILTERING`, strings.Join(keyConditions, " AND "))
	} else {
		query = `SELECT name, time FROM logs.events 
			WHERE project_id = ? 
			ORDER BY time DESC ALLOW FILTERING`
	}

	args = append([]interface{}{projectID}, args...)

	iter := c.Session.Query(query, args...).Iter()

	// Group by event name and count
	eventMap := make(map[string]*models.EventSummary)

	var name string
	var timestamp time.Time

	for iter.Scan(&name, &timestamp) {
		if summary, exists := eventMap[name]; exists {
			summary.Count++
			if timestamp.After(summary.LastTimestamp) {
				summary.LastTimestamp = timestamp
			}
		} else {
			eventMap[name] = &models.EventSummary{
				Name:          name,
				LastTimestamp: timestamp,
				Count:         1,
			}
		}
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("[cassandra.GetEventSummaries] Failed to iterate results: %v", err)
	}

	// Convert map to slice
	var summaries []models.EventSummary
	for _, summary := range eventMap {
		summaries = append(summaries, *summary)
	}

	return summaries, nil
}

// GetEventDetails retrieves detailed event data for a specific event name
func (c *CassandraClient) GetEventDetails(projectID, eventName string, filterKeys []string, offset, limit int) ([]models.Event, error) {
	var query string
	var args []interface{}

	// Start with the main parameters
	args = append(args, projectID, eventName)

	if len(filterKeys) > 0 {
		// Build query with key filters
		keyConditions := make([]string, len(filterKeys))
		for i, key := range filterKeys {
			keyConditions[i] = "keys CONTAINS ?"
			args = append(args, key)
		}
		query = fmt.Sprintf(`SELECT event_id, project_id, name, time, keys FROM logs.events 
			WHERE project_id = ? AND name = ? AND %s 
			ORDER BY time DESC 
			LIMIT ? ALLOW FILTERING`, strings.Join(keyConditions, " AND "))
	} else {
		query = `SELECT event_id, project_id, name, time, keys FROM logs.events 
			WHERE project_id = ? AND name = ? 
			ORDER BY time DESC 
			LIMIT ? ALLOW FILTERING`
	}

	// Add the limit parameter at the end
	args = append(args, limit)

	iter := c.Session.Query(query, args...).Iter()

	var events []models.Event
	var eventID gocql.UUID
	var name string
	var timestamp time.Time
	var keys []string

	for iter.Scan(&eventID, &projectID, &name, &timestamp, &keys) {
		event := models.Event{
			EventID:   eventID,
			ProjectID: projectID,
			Name:      name,
			Timestamp: timestamp,
			Keys:      keys,
			CreatedAt: timestamp, // Using timestamp as created_at for now
		}
		events = append(events, event)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("[cassandra.GetEventDetails] Failed to iterate results: %v", err)
	}

	return events, nil
}
