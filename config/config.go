package config

import (
	"log"
	"os"
	"strconv"
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

type CassandraConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Keyspace string
}

type Config struct {
	CockroachDBConfig
	KafkaConfig
	CassandraConfig CassandraConfig
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("[config.getEnv] Environment variable %s not set, using default: %s", key, defaultValue)
	return defaultValue
}

func Load() *Config {
	cassandraPort, err := strconv.Atoi(getEnv("CASSANDRA_PORT", "9042"))
	if err != nil {
		log.Fatalf("[config.Load] Invalid CASSANDRA_PORT: %v", err)
	}

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
		CassandraConfig: CassandraConfig{
			Host:     getEnv("CASSANDRA_HOST", "localhost"),
			Port:     cassandraPort,
			User:     getEnv("CASSANDRA_USER", "cassandra"),
			Password: getEnv("CASSANDRA_PASSWORD", "cassandra"),
			Keyspace: getEnv("CASSANDRA_KEYSPACE", "logs"),
		},
	}
	return cfg
}
