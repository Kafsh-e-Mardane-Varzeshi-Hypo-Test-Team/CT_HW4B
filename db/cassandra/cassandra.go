package cassandra

import (
	"fmt"
	"log"

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
