package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
)

type CockroachClient struct {
	Db *sql.DB
}

func NewCockroachClient(cfg config.CockroachDBConfig) (*CockroachClient, error) {
	connStr := fmt.Sprintf(
		"postgresql://%s@%s:%s/%s?sslmode=disable",
		cfg.User,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("[db.NewCockroachClient] Failed to connect to CockroachDB: %v", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("[db.NewCockroachClient] Failed to ping CockroachDB: %v", err)
	}

	log.Println("[db.NewCockroachClient] Successfully connected to CockroachDB!")
	return &CockroachClient{Db: db}, nil
}

func (c *CockroachClient) LoadSchema(cfg config.CockroachDBConfig) error {
	_, err := c.Db.Exec("CREATE DATABASE IF NOT EXISTS" + cfg.Database)
	if err != nil {
		return fmt.Errorf("[db.LoadSchema] Failed to create database: %v", err)
	}

	_, err = c.Db.Exec(`CREATE TABLE IF NOT EXISTS users (
							id UUID PRIMARY KEY,
							username STRING UNIQUE,
							password STRING
						);

						CREATE TABLE IF NOT EXISTS projects (
							id UUID PRIMARY KEY,
							name STRING,
							user_id UUID REFERENCES users(id),
							api_key STRING UNIQUE,
							searchable_keys STRING[],
							ttl INTERVAL
						);
						`)
	if err != nil {
		return fmt.Errorf("[db.LoadSchema] Failed to create tables: %v", err)
	}
	return nil
}
