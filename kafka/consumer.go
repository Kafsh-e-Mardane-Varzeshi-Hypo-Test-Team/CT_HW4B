package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/models"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
	insert func(event models.LogRequest) error
}

func NewConsumer(cfg config.KafkaConfig, insert func(event models.LogRequest) error) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{cfg.Broker},
		Topic:     cfg.Topic,
		Partition: 0,
		MinBytes:  1,    // 1 byte
		MaxBytes:  10e6, // 10MB
	})

	log.Printf("[kafka.NewConsumer] Successfully connected to Kafka! Broker: %s, Topic: %s", cfg.Broker, cfg.Topic)
	return &Consumer{reader: reader, insert: insert}
}

func (c *Consumer) ConsumeMessages() {
	for {
		m, err := c.reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("[kafka.ConsumeMessages] Failed to read message: %v", err)
			continue
		}

		var event models.LogRequest
		if err := json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("[kafka.ConsumeMessages] Failed to unmarshal message: %v", err)
			continue
		}

		if err := c.insert(event); err != nil {
			log.Printf("[kafka.ConsumeMessages] Failed to insert event: %v", err)
		} else {
			log.Printf("[kafka.ConsumeMessages] Successfully inserted event: %s", event.Payload.Name)
		}
	}
}
