#!/bin/sh
set -eu

API_URL=${API_URL:-http://localhost:8083}
API_TOKEN=${API_TOKEN:-local-architecture-token-0123456789abcdef}
GRAPH_ID=${GRAPH_ID:-insight-track}

cleanup() {
  status=$?
  trap - EXIT INT TERM
  if [ "$status" -ne 0 ]; then
    docker compose logs --no-color api neo4j
  fi
  docker compose down --volumes
  exit "$status"
}
trap cleanup EXIT INT TERM

authorized_curl() {
  curl --fail --silent --show-error \
    -H "Authorization: Bearer $API_TOKEN" \
    "$@"
}

json_assert() {
  python3 -c "import json,sys; value=json.load(sys.stdin); assert $1, value"
}

echo "Starting architecture graph stack"
docker compose up --build --detach

echo "Waiting for the API and Neo4j"
attempt=0
until curl --fail --silent "$API_URL/health/ready" >/dev/null 2>&1; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 90 ]; then
    echo "API did not become ready"
    exit 1
  fi
  sleep 2
done

echo "Creating the InsightTrack architecture graph"
authorized_curl -X PUT \
  -H 'Content-Type: application/json' \
  -d '{"name":"InsightTrack","description":"InsightTrack software architecture"}' \
  "$API_URL/v1/graphs/$GRAPH_ID" | json_assert "value['id'] == '$GRAPH_ID'"

echo "Importing the full architecture snapshot"
authorized_curl -X PUT \
  -H 'Content-Type: application/json' \
  --data-binary @seed/insight-track.json \
  "$API_URL/v1/graphs/$GRAPH_ID/snapshot" | json_assert "len(value['nodes']) >= 80 and len(value['relationships']) >= 110"

echo "Tracing frontend to PostgreSQL"
authorized_curl -X POST \
  -H 'Content-Type: application/json' \
  -d '{"question":"Show the path from Project Page to projects table","focus_node_id":"page:project","target_node_id":"table:projects","max_depth":12}' \
  "$API_URL/v1/graphs/$GRAPH_ID/questions" | json_assert "value['intent'] == 'shortest_path' and len(value['evidence']['paths']) == 1"

echo "Finding downstream causes and upstream impact"
authorized_curl -X POST \
  -H 'Content-Type: application/json' \
  -d '{"question":"What can cause errors in Project Page?","focus_node_id":"page:project"}' \
  "$API_URL/v1/graphs/$GRAPH_ID/questions" | json_assert "value['intent'] == 'root_causes' and len(value['evidence']['paths']) > 0"

authorized_curl -X POST \
  -H 'Content-Type: application/json' \
  -d '{"question":"What breaks if Identity Provider fails?","focus_node_id":"external:identity-provider"}' \
  "$API_URL/v1/graphs/$GRAPH_ID/questions" | json_assert "value['intent'] == 'blast_radius' and len(value['evidence']['paths']) > 0"

echo "Checking direct graph traversal and authentication"
authorized_curl "$API_URL/v1/graphs/$GRAPH_ID/nodes/endpoint%3Ainvite-project/dependencies?max_depth=8" \
  | json_assert "len(value['paths']) > 0"

status=$(curl --silent --output /dev/null --write-out '%{http_code}' "$API_URL/v1/graphs")
test "$status" = "401"

echo "Architecture graph smoke test passed"
