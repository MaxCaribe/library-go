.PHONY: server test build fmt vet tidy

server:
	go run ./cmd/server/

test:
	go test ./...