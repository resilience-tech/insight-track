ATTACH MATERIALIZED VIEW _ UUID 'c9172266-f56d-4c55-b47d-d1daba71e355' TO default.otel_traces_trace_id_ts
(
    `TraceId` String,
    `Start` DateTime64(9),
    `End` DateTime64(9)
)
AS SELECT
    TraceId,
    min(Timestamp) AS Start,
    max(Timestamp) AS End
FROM default.otel_traces
WHERE TraceId != ''
GROUP BY TraceId
