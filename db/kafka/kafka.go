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
	var conn *kafka.Conn
	var err error

	for _, broker := range cfg.Brokers {
		conn, err = kafka.Dial("tcp", broker)
		if err == nil {
			break
		}
		log.Printf("[kafka.createTopic] Failed to connect to broker %s: %v", broker, err)
	}

	if conn == nil || err != nil {
		return fmt.Errorf("[kafka.createTopic] All brokers failed: %v", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("[kafka.createTopic] Failed to get controller: %v", err)
	}

	controllerAddr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
	controllerConn, err := kafka.Dial("tcp", controllerAddr)
	if err != nil {
		return fmt.Errorf("[kafka.createTopic] Failed to dial controller (%s): %v", controllerAddr, err)
	}
	defer controllerConn.Close()

	topicConfig := kafka.TopicConfig{
		Topic:             cfg.Topic,
		NumPartitions:     3,
		ReplicationFactor: 3,
	}

	err = controllerConn.CreateTopics(topicConfig)
	if err != nil {
		return fmt.Errorf("[kafka.createTopic] Failed to create topic: %v", err)
	}

	log.Printf("[kafka.createTopic] Successfully created topic: %s", cfg.Topic)
	return nil
}
