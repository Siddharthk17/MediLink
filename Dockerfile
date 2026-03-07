# Stage 1: Builder
FROM golang:1.22-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build API binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/medilink-api ./cmd/api

# Build Worker binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/medilink-worker ./cmd/worker

# Stage 2: Migration runner
FROM golang:1.22-alpine AS migrate
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
COPY migrations /migrations
ENTRYPOINT ["sh", "-c", "migrate -path /migrations -database $DATABASE_URL up"]

# Stage 3: API runtime
FROM gcr.io/distroless/static-debian12 AS api
COPY --from=builder /bin/medilink-api /medilink-api
EXPOSE 8080
ENTRYPOINT ["/medilink-api"]

# Stage 4: Worker runtime
FROM gcr.io/distroless/static-debian12 AS worker
COPY --from=builder /bin/medilink-worker /medilink-worker
ENTRYPOINT ["/medilink-worker"]
