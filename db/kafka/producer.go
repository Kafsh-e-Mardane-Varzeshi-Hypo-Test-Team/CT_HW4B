package kafka

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
	topic  string
}

func NewProducer(cfg config.KafkaConfig) *Producer {
	if len(cfg.Brokers) == 0 {
		log.Fatal("[kafka.NewProducer] No brokers provided in config")
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    1, // Send messages immediately
		RequiredAcks: kafka.RequireOne,
		Async:        false,
		Compression:  kafka.Snappy,
	}

	log.Printf("[kafka.NewProducer] Successfully connected to Kafka! Brokers: %s, Topic: %s", cfg.Brokers, cfg.Topic)
	return &Producer{
		writer: writer,
		topic:  cfg.Topic,
	}
}

func (p *Producer) ProduceMessage(ctx context.Context, message []byte) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := p.writer.WriteMessages(ctx, kafka.Message{
		Value: message,
	})
	if err != nil {
		return fmt.Errorf("[kafka.ProduceMessage] Failed to write message to Kafka: %v", err)
	}

	return nil
}

func (p *Producer) Close() error {
	log.Printf("[kafka.Producer] Closing writer for topic=%q", p.topic)
	return p.writer.Close()
}
