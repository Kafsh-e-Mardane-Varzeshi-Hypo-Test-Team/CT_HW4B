CREATE KEYSPACE IF NOT EXISTS logs 
WITH replication = {
  'class': 'SimpleStrategy',
  'replication_factor': 1
};

USE logs;

-- Main events table 
CREATE TABLE IF NOT EXISTS logs.events (
    event_id UUID,
    project_id UUID,
    name TEXT,
    time TIMESTAMP,
    created_at TIMESTAMP,
    keys LIST<TEXT>, 
    PRIMARY KEY ((project_id), time, event_id)
) WITH CLUSTERING ORDER BY (time DESC);
   