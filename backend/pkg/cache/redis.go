// Package cache provides Redis client setup using go-redis.
package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// RedisClient wraps the go-redis client.
type RedisClient struct {
	Client *redis.Client
}

// NewRedisClient creates a new Redis client from a connection URL.
func NewRedisClient(url string) (*RedisClient, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	log.Info().Msg("Redis connection established")

	return &RedisClient{Client: client}, nil
}

// Close closes the Redis connection.
func (r *RedisClient) Close() error {
	log.Info().Msg("Redis connection closed")
	return r.Client.Close()
}

// HealthCheck verifies the Redis connection is alive.
func (r *RedisClient) HealthCheck() error {
	return r.Client.Ping(context.Background()).Err()
}
