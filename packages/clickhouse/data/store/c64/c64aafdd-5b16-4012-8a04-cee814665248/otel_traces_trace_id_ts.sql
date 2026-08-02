ATTACH TABLE _ UUID '431c61e0-e549-48cb-90af-794925806447'
(
    `TraceId` String CODEC(ZSTD(1)),
    `Start` DateTime CODEC(Delta(4), ZSTD(1)),
    `End` DateTime CODEC(Delta(4), ZSTD(1)),
    INDEX idx_trace_id TraceId TYPE bloom_filter(0.01) GRANULARITY 1
)
ENGINE = MergeTree
PARTITION BY toDate(Start)
ORDER BY (TraceId, Start)
TTL toDateTime(Start) + toIntervalDay(30)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1
