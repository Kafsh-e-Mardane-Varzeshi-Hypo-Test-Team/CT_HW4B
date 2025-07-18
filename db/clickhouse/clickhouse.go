package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/models"
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
	query := `
	INSERT INTO events (
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
		event.Payload.Keys,
	)
	if err != nil {
		return fmt.Errorf("[clickhouse.Insert] Failed to insert event: %v", err)
	}
	log.Printf("[clickhouse.Insert] Successfully inserted event: %+v", event)
	return nil
}
