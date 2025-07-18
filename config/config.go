package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type CockroachDBConfig struct {
	Host     string
	Ports    []int
	Port     string // Keep for backward compatibility
	User     string
	Database string
}

type KafkaConfig struct {
	Broker string
	Topic  string
}

type CassandraConfig struct {
	Host           string
	Ports          []int
	Port           int // Keep for backward compatibility
	User           string
	Password       string
	Keyspace       string
	Consistency    string
	Timeout        int
	NumConns       int
	ConnectTimeout int
	QueryTimeout   int
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

	// Parse CockroachDB ports from environment variable
	cockroachPorts := getEnv("COCKROACHDB_PORTS", "26257,26258,26259")
	portStrings := strings.Split(cockroachPorts, ",")
	var cockroachPortsList []int
	for _, portStr := range portStrings {
		port, err := strconv.Atoi(strings.TrimSpace(portStr))
		if err != nil {
			log.Printf("[config.Load] Invalid port in COCKROACHDB_PORTS: %s, skipping", portStr)
			continue
		}
		cockroachPortsList = append(cockroachPortsList, port)
	}

	// Parse Cassandra ports from environment variable
	cassandraPortsStr := getEnv("CASSANDRA_PORTS", "9042,9043,9044")
	cassandraPortStrings := strings.Split(cassandraPortsStr, ",")
	var cassandraPorts []int
	for _, portStr := range cassandraPortStrings {
		port, err := strconv.Atoi(strings.TrimSpace(portStr))
		if err != nil {
			log.Printf("[config.Load] Invalid port in CASSANDRA_PORTS: %s, skipping", portStr)
			continue
		}
		cassandraPorts = append(cassandraPorts, port)
	}

	cfg := &Config{
		CockroachDBConfig: CockroachDBConfig{
			Host:     getEnv("COCKROACHDB_HOST", "localhost"),
			Ports:    cockroachPortsList,
			Port:     getEnv("COCKROACHDB_PORT", "26257"),
			User:     getEnv("COCKROACHDB_USER", "root"),
			Database: getEnv("COCKROACHDB_DATABASE", "logs"),
		},
		KafkaConfig: KafkaConfig{
			Broker: getEnv("KAFKA_BROKER", "localhost:9092"),
			Topic:  getEnv("KAFKA_TOPIC", "raw_logs"),
		},
		CassandraConfig: CassandraConfig{
			Host:           getEnv("CASSANDRA_HOST", "localhost"),
			Ports:          cassandraPorts,
			Port:           cassandraPort,
			User:           getEnv("CASSANDRA_USER", "cassandra_user"),
			Password:       getEnv("CASSANDRA_PASSWORD", "cassandra_password"),
			Keyspace:       getEnv("CASSANDRA_KEYSPACE", "logs"),
			Consistency:    getEnv("CASSANDRA_CONSISTENCY", "quorum"),
			Timeout:        getTimeout(getEnv("CASSANDRA_TIMEOUT", "5")),
			NumConns:       getNumConns(getEnv("CASSANDRA_NUM_CONNS", "10")),
			ConnectTimeout: getTimeout(getEnv("CASSANDRA_CONNECT_TIMEOUT", "5")),
			QueryTimeout:   getTimeout(getEnv("CASSANDRA_QUERY_TIMEOUT", "5")),
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

func getTimeout(value string) int {
	timeout, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("[config.getTimeout] Invalid timeout value: %s, using default: 5", value)
		return 5
	}
	return timeout
}

func getNumConns(value string) int {
	numConns, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("[config.getNumConns] Invalid num_conns value: %s, using default: 10", value)
		return 10
	}
	return numConns
}
