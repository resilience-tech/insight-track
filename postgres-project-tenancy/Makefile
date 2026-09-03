.PHONY: build test vet up down logs smoke

build:
	mkdir -p bin
	go build -o bin/project-tenancy ./cmd/api

test:
	go test ./...

vet:
	go vet ./...

up:
	docker compose up --build --detach

down:
	docker compose down

logs:
	docker compose logs --follow api postgres keycloak

smoke:
	./scripts/smoke.sh
