package cassandra

import (
	"fmt"
	"log"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
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

func (c *CassandraClient) LoadSchema() error {
	err := c.Session.Query(`
		CREATE KEYSPACE IF NOT EXISTS logs WITH REPLICATION = {
			'class': 'SimpleStrategy',
			'replication_factor': 1
		};
	`).Exec()
	if err != nil {
		return fmt.Errorf("[cassandra.LoadSchema] Failed to create keyspace: %v", err)
	}

	err = c.Session.Query(`
		CREATE TABLE IF NOT EXISTS logs.events (
			event_id UUID,
			project_id UUID,
			name TEXT,
			time TIMESTAMP,
			keys LIST<TEXT>, 
			PRIMARY KEY ((project_id), time, event_id)
		) WITH CLUSTERING ORDER BY (time DESC)
		AND default_time_to_live = 2592000;
		);
	`).Exec()
	if err != nil {
		return fmt.Errorf("[cassandra.LoadSchema] Failed to create table: %v", err)
	}

	err = c.Session.Query(`
		-- Optimized view for key searches
		CREATE MATERIALIZED VIEW events_by_key AS
		SELECT event_id, project_id, name, time, keys
		FROM events
		WHERE project_id IS NOT NULL 
		AND time IS NOT NULL 
		AND event_id IS NOT NULL
		AND key IS NOT NULL
		PRIMARY KEY ((project_id, key), time, event_id)
		WITH CLUSTERING ORDER BY (time DESC);
	`).Exec()
	if err != nil {
		return fmt.Errorf("[cassandra.LoadSchema] Failed to create materialized view: %v", err)
	}

	log.Println("[cassandra.LoadSchema] Cassandra schema loaded successfully!")
	return nil
}