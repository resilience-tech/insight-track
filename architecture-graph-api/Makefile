.PHONY: check compose-down compose-logs compose-up seed smoke test

API_URL ?= http://localhost:8083
API_TOKEN ?= local-architecture-token-0123456789abcdef

check:
	node --check src/*.ts
	node --test test/*.test.ts

test:
	node --test test/*.test.ts

compose-up:
	docker compose up --build --detach

compose-down:
	docker compose down

compose-logs:
	docker compose logs --follow api neo4j

seed:
	curl --fail --silent --show-error -X PUT \
		-H "Authorization: Bearer $(API_TOKEN)" \
		-H "Content-Type: application/json" \
		-d '{"name":"InsightTrack","description":"InsightTrack software architecture"}' \
		"$(API_URL)/v1/graphs/insight-track"
	curl --fail --silent --show-error -X PUT \
		-H "Authorization: Bearer $(API_TOKEN)" \
		-H "Content-Type: application/json" \
		--data-binary @seed/insight-track.json \
		"$(API_URL)/v1/graphs/insight-track/snapshot"

smoke:
	./scripts/smoke.sh
