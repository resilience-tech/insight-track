CREATE DATABASE IF NOT EXISTS insight_track;

CREATE TABLE IF NOT EXISTS insight_track.metrics
(
    id UUID,
    project_id UUID,
    service_id UUID,
    timestamp DateTime64(3, 'UTC'),
    name LowCardinality(String),
    value Float64,
    unit LowCardinality(String),
    attributes Map(String, String),
    ingested_at DateTime64(3, 'UTC')
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (project_id, service_id, name, timestamp, id)
TTL timestamp + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192;

CREATE TABLE IF NOT EXISTS insight_track.logs
(
    id UUID,
    project_id UUID,
    service_id UUID,
    timestamp DateTime64(3, 'UTC'),
    severity LowCardinality(String),
    message String,
    trace_id String,
    span_id String,
    attributes Map(String, String),
    ingested_at DateTime64(3, 'UTC'),
    INDEX logs_trace_id_idx trace_id TYPE bloom_filter(0.01) GRANULARITY 4
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (project_id, service_id, timestamp, id)
TTL timestamp + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192;

CREATE TABLE IF NOT EXISTS insight_track.spans
(
    id UUID,
    project_id UUID,
    service_id UUID,
    trace_id String,
    span_id String,
    parent_span_id String,
    name LowCardinality(String),
    start_time DateTime64(3, 'UTC'),
    duration_ms Float64,
    status LowCardinality(String),
    attributes Map(String, String),
    ingested_at DateTime64(3, 'UTC'),
    INDEX spans_trace_id_idx trace_id TYPE bloom_filter(0.01) GRANULARITY 4
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(start_time)
ORDER BY (project_id, service_id, start_time, id)
TTL start_time + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192;
