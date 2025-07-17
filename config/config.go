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

type ClickHouseConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

type Config struct {
	CockroachDBConfig
	KafkaConfig
	CassandraConfig
	ClickHouseConfig
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

	clickhousePort, err := strconv.Atoi(getEnv("CLICKHOUSE_PORT", "9000"))
	if err != nil {
		log.Fatalf("[config.Load] Invalid CLICKHOUSE_PORT: %v", err)
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
			User:     getEnv("CASSANDRA_USER", "cassandra_user"),
			Password: getEnv("CASSANDRA_PASSWORD", "cassandra_password"),
			Keyspace: getEnv("CASSANDRA_KEYSPACE", "logs"),
		},
		ClickHouseConfig: ClickHouseConfig{
			Host:     getEnv("CLICKHOUSE_HOST", "localhost"),
			Port:     clickhousePort,
			Username: getEnv("CLICKHOUSE_USERNAME", "clickhouse_user"),
			Password: getEnv("CLICKHOUSE_PASSWORD", "clickhouse_password"),
			Database: getEnv("CLICKHOUSE_DATABASE", "logs"),
		},
	}
	return cfg
}
