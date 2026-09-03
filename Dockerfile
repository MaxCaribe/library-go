FROM golang:1.26-alpine AS builder

ENV CGO_ENABLED=0
ENV GOOS=linux

RUN apk --no-cache add ca-certificates git

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# -s: Omit the symbol table.
# -w: Omit the DWARF debug information.
RUN mkdir bin && \
    for dir in ./cmd/*/; do \
        name=$(basename "$dir"); \
        go build -ldflags="-s -w" -o bin/$name "$dir"; \
    done


FROM alpine:3.22

RUN addgroup -S group && adduser -S user -G group

USER user

WORKDIR /app

COPY --from=builder --chown=user:group /build/bin/ /app/

EXPOSE 8080

CMD ["./server"]
