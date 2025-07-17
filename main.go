package main

import (
	"log"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db"
)

func main() {
	cfg := config.Load()

	cockroach, err := db.NewCockroachClient(cfg.CockroachDBConfig)
	if err != nil {
		log.Fatalf("[main] Failed to create CockroachDB client: %v", err)
	}

	err = cockroach.LoadSchema(cfg.CockroachDBConfig)
	if err != nil {
		log.Fatalf("[main] Failed to load db schema: %v", err)
	}
}
