package kafka

import (
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/Kafsh-e-Mardane-Varzeshi-Hypo-Test-Team/CT_HW4B/config"
	"github.com/segmentio/kafka-go"
)

func CreateTopic(cfg config.KafkaConfig) error {
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
