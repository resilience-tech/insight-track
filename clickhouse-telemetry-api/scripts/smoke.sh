#!/bin/sh
set -eu

API_URL=${API_URL:-http://localhost:8082}
API_TOKEN=${API_TOKEN:-local-development-token-0123456789abcdef}
PROJECT_ID=${PROJECT_ID:-019c3d56-7890-7abc-8def-0123456789ab}
SERVICE_ID=${SERVICE_ID:-019c3d56-7890-7abc-8def-0123456789ac}

cleanup() {
  status=$?
  trap - EXIT INT TERM
  if [ "$status" -ne 0 ]; then
    docker compose logs --no-color api clickhouse
  fi
  docker compose down --volumes
  exit "$status"
}
trap cleanup EXIT INT TERM

json_field() {
  python3 -c 'import json,sys; print(json.load(sys.stdin)[sys.argv[1]])' "$1"
}

json_length() {
  python3 -c 'import json,sys; print(len(json.load(sys.stdin)[sys.argv[1]]))' "$1"
}

authorized_curl() {
  curl --fail --silent --show-error \
    -H "Authorization: Bearer $API_TOKEN" \
    "$@"
}

echo "Starting ClickHouse telemetry stack"
docker compose up --build --detach

echo "Waiting for the API"
attempt=0
until curl --fail --silent "$API_URL/health/ready" >/dev/null 2>&1; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 60 ]; then
    echo "API did not become ready"
    exit 1
  fi
  sleep 2
done

timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
base="$API_URL/v1/projects/$PROJECT_ID/services/$SERVICE_ID/telemetry"

echo "Writing metric, log, and trace batches"
metric_response=$(authorized_curl \
  -H 'Content-Type: application/json' \
  -d "{\"items\":[{\"timestamp\":\"$timestamp\",\"name\":\"http.requests\",\"value\":1,\"unit\":\"request\",\"attributes\":{\"method\":\"GET\"}}]}" \
  "$base/metrics")
test "$(printf '%s' "$metric_response" | json_field accepted)" = "1"

log_response=$(authorized_curl \
  -H 'Content-Type: application/json' \
  -d "{\"items\":[{\"timestamp\":\"$timestamp\",\"severity\":\"info\",\"message\":\"smoke test\",\"trace_id\":\"0123456789abcdef0123456789abcdef\",\"span_id\":\"0123456789abcdef\"}]}" \
  "$base/logs")
test "$(printf '%s' "$log_response" | json_field accepted)" = "1"

span_response=$(authorized_curl \
  -H 'Content-Type: application/json' \
  -d "{\"items\":[{\"trace_id\":\"0123456789abcdef0123456789abcdef\",\"span_id\":\"0123456789abcdef\",\"name\":\"GET /health\",\"start_time\":\"$timestamp\",\"duration_ms\":4.5,\"status\":\"ok\"}]}" \
  "$base/traces")
test "$(printf '%s' "$span_response" | json_field accepted)" = "1"

echo "Reading telemetry back"
test "$(authorized_curl "$base/metrics?name=http.requests" | json_length items)" = "1"
test "$(authorized_curl "$base/logs?severity=info" | json_length items)" = "1"
test "$(authorized_curl "$base/traces?status=ok" | json_length items)" = "1"
test "$(authorized_curl "$base/summary" | json_field metric_points)" = "1"

echo "Checking authentication"
status=$(curl --silent --output /dev/null --write-out '%{http_code}' "$base/metrics")
test "$status" = "401"

echo "Smoke test passed"
