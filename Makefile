.PHONY: server test test-integration cover generate build fmt vet tidy db-up db-down migrate-up migrate-down migrate-status create-migration

server:
	go run ./cmd/server/

test:
	go test ./...

# Integration tests start a Postgres container unless TEST_DATABASE_URL points
# at one already running. Without Docker or that variable, they skip.
test-integration:
	go test ./... -count=1

# -coverpkg is required, not cosmetic: test/integration exercises application
# and repositories from outside those directories, so plain -cover reports them
# as untested.
cover:
	go test -coverpkg=./internal/... -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1
	@echo "html report: go tool cover -html=coverage.out"

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
