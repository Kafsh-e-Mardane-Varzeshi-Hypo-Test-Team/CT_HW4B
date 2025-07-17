package main

import (
	"log"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/api"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db/cassandra"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db/clickhouse"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db/cockroach"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/db/kafka"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	cockroach, err := cockroach.NewCockroachClient(cfg.CockroachDBConfig)
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
	kafkaConsumerCassandra := kafka.NewConsumer(cfg.KafkaConfig, cassandra.Insert)
	go kafkaConsumerCassandra.ConsumeMessages()

    clickhouse, err := clickhouse.NewClickHouseClient(cfg.ClickHouseConfig)
    if err != nil {
        log.Fatalf("[main] Failed to create ClickHouse client: %v", err)
    }
    kafkaConsumerClickHouse := kafka.NewConsumer(cfg.KafkaConfig, clickhouse.Insert)
    go kafkaConsumerClickHouse.ConsumeMessages()

	handler := api.NewHandler(cockroach, kafkaProducer, cassandra)
	r := gin.Default()
	r.POST("/api/logs", handler.SubmitLogHandler)
	log.Fatalf("[main] Error while running gin router: %v", r.Run(":9090"))
}
