.PHONY: server test generate build fmt vet tidy

server:
	go run ./cmd/server/

test:
	go test ./...

generate:
	go generate ./...
