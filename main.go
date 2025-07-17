package main

import (
	"log"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/api"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/cassandra"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/kafka"
	"github.com/gin-gonic/gin"
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

	err = kafka.CreateTopic(cfg.KafkaConfig)
	if err != nil {
		log.Fatalf("[main] Failed to create Kafka topic: %v", err)
	}
	kafkaProducer := kafka.NewProducer(cfg.KafkaConfig)

	cassandra, err := cassandra.NewCassandraClient(cfg.CassandraConfig)
	if err != nil {
		log.Fatalf("[main] Failed to create Cassandra client: %v", err)
	}
	err = cassandra.LoadSchema()
	if err != nil {
		log.Fatalf("[main] Failed to load Cassandra schema: %v", err)
	}

	handler := api.NewHandler(cockroach, kafkaProducer, cassandra)
	r := gin.Default()
	r.POST("/api/logs", handler.SubmitLogHandler)
	log.Fatalf("[main] Error while running gin router: %v", r.Run(":9000"))
}
