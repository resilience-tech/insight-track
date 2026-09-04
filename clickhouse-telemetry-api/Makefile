.PHONY: build test vet fmt-check up down logs smoke

build:
	mkdir -p bin
	go build -o bin/clickhouse-telemetry-api ./cmd/api

test:
	go test ./...

vet:
	go vet ./...

fmt-check:
	test -z "$$(gofmt -l .)"

up:
	docker compose up --build --detach

down:
	docker compose down

logs:
	docker compose logs --follow api clickhouse

smoke:
	./scripts/smoke.sh
