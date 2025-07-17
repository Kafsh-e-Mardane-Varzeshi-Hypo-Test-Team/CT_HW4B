package config

import (
	"log"
	"os"
)

type CockroachDBConfig struct {
	Host     string
	Port     string
	User     string
	Database string
}

type KafkaConfig struct {
	Broker string
	Topic  string
}

type Config struct {
	CockroachDBConfig
	KafkaConfig
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("[config.getEnv] Environment variable %s not set, using default: %s", key, defaultValue)
	return defaultValue
}

func Load() *Config {
	cfg := &Config{
		CockroachDBConfig: CockroachDBConfig{
			Host:     getEnv("COCKROACHDB_HOST", "localhost"),
			Port:     getEnv("COCKROACHDB_PORT", "26257"),
			User:     getEnv("COCKROACHDB_USER", "root"),
			Database: getEnv("COCKROACHDB_DATABASE", "logs"),
		},
		KafkaConfig: KafkaConfig{
			Broker: getEnv("KAFKA_BROKER", "localhost:9092"),
			Topic:  getEnv("KAFKA_TOPIC", "raw_logs"),
		},
	}
	return cfg
}