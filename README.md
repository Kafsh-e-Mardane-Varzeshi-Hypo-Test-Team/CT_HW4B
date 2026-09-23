# Log Analysis System

A distributed log analysis platform for ingesting, storing, searching, and analyzing application events.

## Overview

The system uses a Go backend with a lightweight HTML/CSS/JavaScript frontend. Event ingestion is handled asynchronously through Kafka, with Cassandra and ClickHouse serving different event storage and querying workloads. CockroachDB stores users and project metadata.

```text
Frontend (HTML/CSS/JS)
          │
          ▼
      Go + Gin
       /     \
      /       \
CockroachDB   Kafka
                │
          ┌─────┴─────┐
          ▼           ▼
      Cassandra    ClickHouse
```

## Features

* Project and user management
* Event ingestion through an HTTP API
* Asynchronous event processing through Kafka
* Event storage and time-based retrieval with Cassandra
* Event filtering and analytics with ClickHouse
* Project-specific searchable keys
* Configurable event retention

## Tech Stack

| Component              | Technology                                         |
| ---------------------- | -------------------------------------------------- |
| Frontend               | HTML, CSS, JavaScript                              |
| Backend                | [Go](https://go.dev/)                              |
| HTTP Router            | [Gin](https://gin-gonic.com/)                      |
| Message Broker         | [Apache Kafka](https://kafka.apache.org/)          |
| Transactional Database | [CockroachDB](https://www.cockroachlabs.com/docs/) |
| Event Store            | [Apache Cassandra](https://cassandra.apache.org/)  |
| Analytics Database     | [ClickHouse](https://clickhouse.com/docs/)         |
| Infrastructure         | [Docker Compose](https://docs.docker.com/compose/) |

## Architecture

The Go backend receives requests from the frontend and handles user/project operations directly through CockroachDB.

Events are published to Kafka and consumed independently by Cassandra and ClickHouse. Cassandra is used for event retrieval, while ClickHouse is used for filtering and analytical queries.

This separation allows event ingestion and analytical workloads to be handled independently.

## Getting Started

### Prerequisites

Make sure the following are installed:

* [Go](https://go.dev/)
* [Docker](https://docs.docker.com/get-docker/)
* [Docker Compose](https://docs.docker.com/compose/)

### Start the infrastructure

From the project directory, start the required services:

```bash
docker compose up -d
```

Cassandra requires additional initialization time before the backend can be started.

Check the service status with:

```bash
docker compose ps
```

You can also monitor the services continuously with:

```bash
watch -n1 docker compose ps
```

Wait until the `cassandra-init` service has finished and is no longer running.

### Start the backend

Download the Go dependencies:

```bash
go mod download
```

Then start the application:

```bash
go run .
```

The application will be available at:

```text
http://localhost:9090
```
