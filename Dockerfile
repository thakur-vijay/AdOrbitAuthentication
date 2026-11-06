FROM golang:1.27.1-alpine3.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# Install Swagger CLI
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Generate Swagger documentation
RUN swag init -g cmd/server/main.go -o docs --parseInternal

# Build application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/server ./cmd/server

FROM alpine:3.24

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/server ./server

COPY --from=builder /app/internal/database/migrations ./internal/database/migrations

# Copy generated Swagger documentation
COPY --from=builder /app/docs ./docs

EXPOSE 8080

CMD ["./server"]