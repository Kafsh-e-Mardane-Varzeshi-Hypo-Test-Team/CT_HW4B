package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/models"
	"github.com/gocql/gocql"
)

type ClickHouseClient struct {
	DB *sql.DB
}

func NewClickHouseClient(cfg config.ClickHouseConfig) (*ClickHouseClient, error) {
	connStr := fmt.Sprintf("tcp://%s:%d?username=%s&password=%s&database=%s",
		cfg.Host,
		cfg.Port,
		cfg.Username,
		cfg.Password,
		cfg.Database,
	)

	db, err := sql.Open("clickhouse", connStr)
	if err != nil {
		return nil, fmt.Errorf("[clickhouse.NewClickHouseClient] Failed to connect to ClickHouse: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("[clickhouse.NewClickHouseClient] Failed to ping ClickHouse: %v", err)
	}

	log.Println("[clickhouse.NewClickHouseClient] Successfully connected to ClickHouse!")
	return &ClickHouseClient{DB: db}, nil
}

func (c *ClickHouseClient) Insert(event models.LogRequest) error {
	// Extract keys from the data map
	keys := make([]string, 0, len(event.Payload.Data))
	for key := range event.Payload.Data {
		keys = append(keys, key)
	}

	query := `
	INSERT INTO logs.events (
		event_id,
		project_id,
		name,
		time,
		keys
	) VALUES (?, ?, ?, ?, ?)`

	_, err := c.DB.ExecContext(context.Background(), query,
		event.EventID,
		event.ProjectID,
		event.Payload.Name,
		event.Payload.Timestamp,
		keys,
	)
	if err != nil {
		return fmt.Errorf("[clickhouse.Insert] Failed to insert event: %v", err)
	}
	log.Printf("[clickhouse.Insert] Successfully inserted event: %+v", event)
	return nil
}

// GetEventSummaries retrieves aggregated event data for a project using ClickHouse
func (c *ClickHouseClient) GetEventSummaries(projectID string, filterKeys []string) ([]models.EventSummary, error) {
	var args []interface{}

	baseQuery := `
		SELECT 
			name,
			COUNT(*) as count,
			MAX(time) as last_timestamp
		FROM logs.events 
		WHERE project_id = ?`

	args = append(args, projectID)

	if len(filterKeys) > 0 {
		// Build key filter conditions - use arrayExists for better compatibility
		keyConditions := make([]string, len(filterKeys))
		for i, key := range filterKeys {
			keyConditions[i] = "arrayExists(x -> x = ?, keys)"
			args = append(args, key)
		}
		baseQuery += " AND " + strings.Join(keyConditions, " AND ")
	}

	baseQuery += " GROUP BY name ORDER BY last_timestamp DESC"

	rows, err := c.DB.QueryContext(context.Background(), baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("[clickhouse.GetEventSummaries] Failed to execute query: %v", err)
	}
	defer rows.Close()

	var summaries []models.EventSummary
	for rows.Next() {
		var summary models.EventSummary
		err := rows.Scan(&summary.Name, &summary.Count, &summary.LastTimestamp)
		if err != nil {
			return nil, fmt.Errorf("[clickhouse.GetEventSummaries] Failed to scan row: %v", err)
		}
		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("[clickhouse.GetEventSummaries] Error iterating rows: %v", err)
	}

	return summaries, nil
}

func (c *ClickHouseClient) GetEventCount(projectID, eventName string, filterKeys []string) (int, error) {
	var args []interface{}
	query := `
		SELECT 
			count()
		FROM logs.events 
		WHERE project_id = ? AND name = ?`
	args = append(args, projectID, eventName)

	if len(filterKeys) > 0 {
		keyConditions := make([]string, len(filterKeys))
		for i, key := range filterKeys {
			keyConditions[i] = "arrayExists(x -> x = ?, keys)"
			args = append(args, key)
		}
		query += " AND " + strings.Join(keyConditions, " AND ")
	}

	var count int
	err := c.DB.QueryRowContext(context.Background(), query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("[clickhouse.GetEventCount] Failed to query count: %v", err)
	}

	return count, nil
}

// GetEventDetails retrieves detailed event data for a specific event name using ClickHouse
func (c *ClickHouseClient) GetEventDetails(projectID, eventName string, filterKeys []string, offset, limit int) ([]models.Event, error) {
	var args []interface{}

	baseQuery := `
		SELECT 
			event_id,
			project_id,
			name,
			time,
			keys
		FROM logs.events 
		WHERE project_id = ? AND name = ?`

	args = append(args, projectID, eventName)

	if len(filterKeys) > 0 {
		// Build key filter conditions - use arrayExists for better compatibility
		keyConditions := make([]string, len(filterKeys))
		for i, key := range filterKeys {
			keyConditions[i] = "arrayExists(x -> x = ?, keys)"
			args = append(args, key)
		}
		baseQuery += " AND " + strings.Join(keyConditions, " AND ")
	}

	baseQuery += " ORDER BY time DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := c.DB.QueryContext(context.Background(), baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("[clickhouse.GetEventDetails] Failed to execute query: %v", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var event models.Event
		var eventIDStr string
		var keys []string
		err := rows.Scan(&eventIDStr, &event.ProjectID, &event.Name, &event.Timestamp, &keys)
		if err != nil {
			return nil, fmt.Errorf("[clickhouse.GetEventDetails] Failed to scan row: %v", err)
		}

		// Convert string UUID to gocql.UUID
		eventID, err := gocql.ParseUUID(eventIDStr)
		if err != nil {
			return nil, fmt.Errorf("[clickhouse.GetEventDetails] Failed to parse UUID: %v", err)
		}

		// Create a minimal data map from keys (since ClickHouse doesn't store full data)
		data := make(map[string]string)
		for _, key := range keys {
			data[key] = "" // Empty value since we don't have the actual data
		}

		event.EventID = eventID
		event.Data = data
		event.CreatedAt = event.Timestamp // Using timestamp as created_at for now
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("[clickhouse.GetEventDetails] Error iterating rows: %v", err)
	}

	return events, nil
}
