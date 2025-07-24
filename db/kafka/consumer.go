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
	topic  string
}

func NewConsumer(cfg config.KafkaConfig, insert func(event models.LogRequest) error, groupId string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		Topic:       cfg.Topic,
		GroupID:     groupId,
		StartOffset: kafka.FirstOffset,
		MinBytes:    1,    // 1 byte
		MaxBytes:    10e6, // 10MB
	})

	log.Printf("[kafka.NewConsumer] Created Kafka consumer for topic=%q, groupID=%q, brokers=%v", cfg.Topic, groupId, cfg.Brokers)
	return &Consumer{
		reader: reader,
		insert: insert,
		topic:  cfg.Topic,
	}
}

func (c *Consumer) ConsumeMessages(ctx context.Context) error {
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("[kafka.ConsumeMessages] Stopping consumer due to context cancellation")
				return nil
			}
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

func (c *Consumer) Close() error {
	log.Printf("[kafka.Consumer] Closing consumer for topic=%q", c.topic)
	return c.reader.Close()
}
