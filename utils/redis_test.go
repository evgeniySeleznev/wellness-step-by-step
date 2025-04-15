package utils

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisOperations(t *testing.T) {
	// Используем тестовый Redis (можно использовать testcontainers в реальном проекте)
	client, err := NewRedisClient()
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"
	expiration := 1 * time.Second

	// Test Set
	if err := client.SetToCache(ctx, key, value, expiration); err != nil {
		t.Errorf("SetToCache failed: %v", err)
	}

	// Test Get
	got, err := client.GetFromCache(ctx, key)
	if err != nil {
		t.Errorf("GetFromCache failed: %v", err)
	}
	if got != value {
		t.Errorf("GetFromCache got = %v, want %v", got, value)
	}

	// Test Expiration
	time.Sleep(2 * time.Second)
	_, err = client.GetFromCache(ctx, key)
	if err == nil {
		t.Error("Expected error after expiration, got nil")
	} else if !errors.Is(err, redis.Nil) {
		t.Errorf("Expected redis.Nil error, got %v", err)
	}
}
