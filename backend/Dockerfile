# Stage 1: Builder
FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build API binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/MediLink-api ./cmd/api

# Build Worker binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/MediLink-worker ./cmd/worker

# Stage 2: Migration runner
FROM golang:1.25-alpine AS migrate
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
COPY migrations /migrations
ENTRYPOINT ["sh", "-c", "migrate -path /migrations -database $DATABASE_URL up"]

# Stage 3: API runtime (debian for Tesseract support)
FROM debian:bookworm-slim AS api
RUN apt-get update && apt-get install -y --no-install-recommends \
    tesseract-ocr \
    tesseract-ocr-eng \
    tesseract-ocr-hin \
    tesseract-ocr-mar \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /bin/MediLink-api /MediLink-api
EXPOSE 8080
ENTRYPOINT ["/MediLink-api"]

# Stage 4: Worker runtime (debian for Tesseract support)
FROM debian:bookworm-slim AS worker
RUN apt-get update && apt-get install -y --no-install-recommends \
    tesseract-ocr \
    tesseract-ocr-eng \
    tesseract-ocr-hin \
    tesseract-ocr-mar \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /bin/MediLink-worker /MediLink-worker
ENTRYPOINT ["/MediLink-worker"]
