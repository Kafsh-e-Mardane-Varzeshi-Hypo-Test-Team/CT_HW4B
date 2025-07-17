package kafka

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
	"github.com/segmentio/kafka-go"
)

type KafkaClient struct {
	writer *kafka.Writer
	reader *kafka.Reader
}

func NewKafkaClient(cfg config.KafkaConfig) (*KafkaClient, error) {
	err := createTopic(cfg)
	if err != nil {
		return nil, fmt.Errorf("[kafka.NewKafkaClient] Failed to create topic: %v", err)
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Broker),
		Topic:        cfg.Topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    1, // Send messages immediately
		RequiredAcks: kafka.RequireOne,
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{cfg.Broker},
		Topic:     cfg.Topic,
		Partition: 0,
		MinBytes:  1,    // 1 byte
		MaxBytes:  10e6, // 10MB
	})

	log.Printf("[kafka.NewKafkaClient] Successfully connected to Kafka! Broker: %s, Topic: %s", cfg.Broker, cfg.Topic)
	return &KafkaClient{writer: writer, reader: reader}, nil
}

func createTopic(cfg config.KafkaConfig) error {
	conn, err := kafka.Dial("tcp", cfg.Broker)
	if err != nil {
		return fmt.Errorf("[kafka.createTopic] Failed to dial Kafka broker: %v", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("[kafka.createTopic] Failed to get controller: %v", err)
	}

	controllerConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return fmt.Errorf("[kafka.createTopic] Failed to dial controller: %v", err)
	}
	defer controllerConn.Close()

	topicConfig := kafka.TopicConfig{
		Topic:             cfg.Topic,
		NumPartitions:     1, // Default partition count
		ReplicationFactor: 1, // Default replication factor
	}

	err = controllerConn.CreateTopics(topicConfig)
	if err != nil {
		return fmt.Errorf("[kafka.createTopic] Failed to create topic: %v", err)
	}

	log.Printf("[kafka.createTopic] Successfully created topic: %s", cfg.Topic)
	return nil
}

func (k *KafkaClient) ProduceMessage(message []byte) error {
	err := k.writer.WriteMessages(context.Background(), kafka.Message{
		Value: message,
	})
	if err != nil {
		return fmt.Errorf("[kafka.ProduceMessage] Failed to write message to Kafka: %v", err)
	}

	return nil
}
