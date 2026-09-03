#!/bin/sh
set -eu

API_URL=${API_URL:-http://localhost:8080}
OIDC_DISCOVERY_URL=${OIDC_DISCOVERY_URL:-http://localhost:8081/realms/project-tenancy/.well-known/openid-configuration}
OIDC_URL=${OIDC_URL:-http://localhost:8081/realms/project-tenancy/protocol/openid-connect/token}

json_field() {
  python3 -c 'import json,sys; print(json.load(sys.stdin)[sys.argv[1]])' "$1"
}

token_for() {
  curl --fail --silent --show-error \
    --data-urlencode 'client_id=project-tenancy-cli' \
    --data-urlencode 'grant_type=password' \
    --data-urlencode "username=$1" \
    --data-urlencode "password=$1" \
    "$OIDC_URL" | json_field access_token
}

echo "Starting local stack"
docker compose up --build --detach

echo "Waiting for API and identity provider"
attempt=0
until curl --fail --silent "$API_URL/health/ready" >/dev/null 2>&1 \
  && curl --fail --silent "$OIDC_DISCOVERY_URL" >/dev/null 2>&1; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 60 ]; then
    docker compose logs api postgres keycloak
    exit 1
  fi
  sleep 2
done

alice_token=$(token_for alice)
bob_token=$(token_for bob)

echo "Creating Alice's project"
project_response=$(curl --fail --silent --show-error \
  -H "Authorization: Bearer $alice_token" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Smoke project","description":"Docker Compose verification"}' \
  "$API_URL/v1/projects")
project_id=$(printf '%s' "$project_response" | json_field id)

echo "Creating a service"
service_response=$(curl --fail --silent --show-error \
  -H "Authorization: Bearer $alice_token" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Web","slug":"web","kind":"http","configuration":{"port":8080}}' \
  "$API_URL/v1/projects/$project_id/services")
service_id=$(printf '%s' "$service_response" | json_field id)

echo "Checking cross-tenant denial before sharing"
status=$(curl --silent --output /dev/null --write-out '%{http_code}' \
  -H "Authorization: Bearer $bob_token" \
  "$API_URL/v1/projects/$project_id/services/$service_id")
test "$status" = "404"

echo "Inviting Bob"
curl --fail --silent --show-error \
  -H "Authorization: Bearer $alice_token" \
  -H 'Content-Type: application/json' \
  -d '{"email":"bob@example.test"}' \
  "$API_URL/v1/projects/$project_id/invitations" >/dev/null

invitation_token=$(docker compose logs --no-color api | python3 -c '
import json,sys
tokens=[]
for line in sys.stdin:
    start=line.find("{")
    if start < 0:
        continue
    try:
        event=json.loads(line[start:])
    except json.JSONDecodeError:
        continue
    if "token" in event:
        tokens.append(event["token"])
if not tokens:
    raise SystemExit("invitation token was not found in local API logs")
print(tokens[-1])')

echo "Accepting the invitation as Bob"
curl --fail --silent --show-error \
  -H "Authorization: Bearer $bob_token" \
  -H 'Content-Type: application/json' \
  -d "{\"token\":\"$invitation_token\"}" \
  "$API_URL/v1/invitations/accept" >/dev/null

echo "Checking shared service access and resource creation"
curl --fail --silent --show-error \
  -H "Authorization: Bearer $bob_token" \
  "$API_URL/v1/projects/$project_id/services/$service_id" >/dev/null
curl --fail --silent --show-error \
  -H "Authorization: Bearer $bob_token" \
  -H 'Content-Type: application/json' \
  -d '{"resource_key":"smoke","payload":{"healthy":true}}' \
  "$API_URL/v1/projects/$project_id/services/$service_id/resources" >/dev/null

echo "Checking that shared members cannot administer project metadata"
status=$(curl --silent --output /dev/null --write-out '%{http_code}' \
  -X PATCH \
  -H "Authorization: Bearer $bob_token" \
  -H 'Content-Type: application/merge-patch+json' \
  -H 'If-Match: "1"' \
  -d '{"name":"Forbidden rename"}' \
  "$API_URL/v1/projects/$project_id")
test "$status" = "403"

echo "Smoke test passed"
