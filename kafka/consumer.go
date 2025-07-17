package kafka

import (
	"context"
	"log"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(cfg config.KafkaConfig) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{cfg.Broker},
		Topic:     cfg.Topic,
		Partition: 0,
		MinBytes:  1,    // 1 byte
		MaxBytes:  10e6, // 10MB
	})

	log.Printf("[kafka.NewConsumer] Successfully connected to Kafka! Broker: %s, Topic: %s", cfg.Broker, cfg.Topic)
	return &Consumer{reader: reader}
}

func (c *Consumer) ConsumeMessages() {
	for {
		m, err := c.reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("[kafka.ConsumeMessages] Failed to read message: %v", err)
			continue
		}
		log.Printf("[kafka.ConsumeMessages] Received message: %s", string(m.Value))
		// TODO: Process the message as needed
	}
}
