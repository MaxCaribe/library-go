.PHONY: server test generate build fmt vet tidy db-up db-down migrate-up migrate-down migrate-status create-migration

server:
	go run ./cmd/server/

test:
	go test ./...

generate:
	go generate ./...

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/server ./cmd/server/

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

migrate-up:
	go run ./cmd/migrate/ up

migrate-down:
	go run ./cmd/migrate/ down

migrate-status:
	go run ./cmd/migrate/ status

create-migration:
	go run ./cmd/migrate/ create $(name)
