.PHONY: run test migrate lint build clean

# Start all services
run:
	docker compose up --build

# Run API only (assumes postgres and redis already running)
run-api:
	go run ./cmd/api/main.go

# Run all tests with race detector
test:
	go test -race -v -count=1 ./...

# Run tests with coverage
test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Apply migrations
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

# Roll back one migration
migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

# Roll back all migrations
migrate-reset:
	migrate -path migrations -database "$(DATABASE_URL)" drop -f

# Lint
lint:
	golangci-lint run ./...

# Security scan
security:
	gosec ./...

# Build binaries
build:
	CGO_ENABLED=0 go build -o bin/medilink-api ./cmd/api
	CGO_ENABLED=0 go build -o bin/medilink-worker ./cmd/worker

# Clean
clean:
	rm -rf bin/ coverage.out coverage.html

# Install dev tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
