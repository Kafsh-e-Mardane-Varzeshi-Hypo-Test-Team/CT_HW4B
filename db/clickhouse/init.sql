CREATE DATABASE IF NOT EXISTS logs;

CREATE TABLE IF NOT EXISTS logs.events (
    event_id UUID,
    project_id UUID,
    name String,
    time DateTime,
    keys Array(String),
    date Date MATERIALIZED toDate(time)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(time)
ORDER BY (project_id, time, event_id);