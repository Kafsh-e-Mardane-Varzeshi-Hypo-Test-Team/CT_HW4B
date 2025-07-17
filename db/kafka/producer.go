package kafka

import (
	"context"
	"fmt"
	"log"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(cfg config.KafkaConfig) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Broker),
		Topic:        cfg.Topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    1, // Send messages immediately
		RequiredAcks: kafka.RequireOne,
	}

	log.Printf("[kafka.NewProducer] Successfully connected to Kafka! Broker: %s, Topic: %s", cfg.Broker, cfg.Topic)
	return &Producer{writer: writer}
}

func (p *Producer) ProduceMessage(message []byte) error {
	err := p.writer.WriteMessages(context.Background(), kafka.Message{
		Value: message,
	})
	if err != nil {
		return fmt.Errorf("[kafka.ProduceMessage] Failed to write message to Kafka: %v", err)
	}

	return nil
}
