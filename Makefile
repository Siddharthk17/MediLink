.PHONY: run test test-unit test-integration migrate lint build clean

# Start all services
run:
	docker compose up --build

# Run API only (assumes postgres and redis already running)
run-api:
	cd backend && go run ./cmd/api/main.go

# Run all tests with race detector
test: test-unit

# Run unit tests only
test-unit:
	cd backend && go test -race -v -count=1 ./tests/unit/...

# Run integration tests (requires live DB, Redis, ES)
test-integration:
	cd backend && go test -race -v -count=1 ./tests/integration/...

# Run all tests (unit + source packages)
test-all:
	cd backend && go test -race -v -count=1 ./...

# Run tests with coverage
test-cover:
	cd backend && go test -race -coverprofile=coverage.out ./...
	cd backend && go tool cover -html=coverage.out -o coverage.html

# Apply migrations
migrate-up:
	cd backend && migrate -path migrations -database "$(DATABASE_URL)" up

# Roll back one migration
migrate-down:
	cd backend && migrate -path migrations -database "$(DATABASE_URL)" down 1

# Roll back all migrations
migrate-reset:
	cd backend && migrate -path migrations -database "$(DATABASE_URL)" drop -f

# Lint
lint:
	cd backend && golangci-lint run ./...

# Security scan
security:
	cd backend && gosec ./...

# Build binaries
build:
	cd backend && CGO_ENABLED=0 go build -o ../bin/medilink-api ./cmd/api
	cd backend && CGO_ENABLED=0 go build -o ../bin/medilink-worker ./cmd/worker

# Clean
clean:
	rm -rf bin/ backend/coverage.out backend/coverage.html

# Install dev tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
