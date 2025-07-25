CREATE DATABASE IF NOT EXISTS logs;

CREATE TABLE IF NOT EXISTS logs.events (
    event_id UUID,
    project_id UUID,
    name String,
    time DateTime,
    keys Array(String),
    created_at DateTime,
    ttl_seconds UInt32,
    date Date MATERIALIZED toDate(time)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(time)
ORDER BY (project_id, time, event_id)
TTL time + INTERVAL ttl_seconds SECOND WHERE ttl_seconds > 0;