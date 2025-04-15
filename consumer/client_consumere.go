package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
	"wellness-step-by-step/step-05/models"
	"wellness-step-by-step/step-05/utils"

	"github.com/segmentio/kafka-go"
)

type ClientEvent struct {
	Event string        `json:"event"`
	Data  models.Client `json:"data"`
}

type ClientConsumer struct {
	repo     models.Repository
	cache    utils.RedisClient
	reader   *kafka.Reader
	shutdown chan struct{}
}

func NewClientConsumer(repo models.Repository, cache utils.RedisClient) *ClientConsumer {
	return &ClientConsumer{
		repo:  repo,
		cache: cache,
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{os.Getenv("KAFKA_BROKER")},
			Topic:   "client_events",
			GroupID: "wellness-group",
			MaxWait: 10 * time.Second,
		}),
		shutdown: make(chan struct{}),
	}
}

func (c *ClientConsumer) Start(ctx context.Context) {
	log.Println("Starting Kafka consumer...")

	go func() {
		for {
			select {
			case <-c.shutdown:
				return
			default:
				c.processMessages(ctx)
			}
		}
	}()
}

func (c *ClientConsumer) Stop() {
	close(c.shutdown)
	if err := c.reader.Close(); err != nil {
		log.Printf("Error closing Kafka reader: %v", err)
	}
}

func (c *ClientConsumer) processMessages(ctx context.Context) {
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		if err == context.Canceled {
			return
		}
		log.Printf("Kafka read error: %v (will retry)", err)
		time.Sleep(5 * time.Second)
		return
	}

	var event ClientEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		log.Printf("Failed to unmarshal Kafka message: %v", err)
		return
	}

	switch event.Event {
	case "client_created":
		c.handleClientCreated(ctx, event.Data)
	case "client_updated":
		c.handleClientUpdated(ctx, event.Data)
	case "client_deleted":
		c.handleClientDeleted(ctx, event.Data.ID)
	default:
		log.Printf("Unknown event type: %s", event.Event)
	}

	// В реальном проекте здесь нужно коммитить offset только после успешной обработки
}

func (c *ClientConsumer) handleClientCreated(ctx context.Context, client models.Client) {
	// 1. Сохраняем в PostgreSQL (если еще не сохранено)
	existing, err := c.repo.GetClientByID(client.ID)
	if err == nil && existing != nil {
		return // Клиент уже существует
	}

	if err := c.repo.CreateClient(&client); err != nil {
		log.Printf("Failed to create client from Kafka: %v", err)
		return
	}

	// 2. Сериализуем клиента в JSON и сохраняем в Redis
	cacheKey := fmt.Sprintf("client:%d", client.ID)
	clientJSON, err := json.Marshal(client)
	if err != nil {
		log.Printf("Failed to marshal client to JSON: %v", err)
		return
	}

	if err := c.cache.SetToCache(ctx, cacheKey, string(clientJSON), 24*time.Hour); err != nil {
		log.Printf("Failed to cache client: %v", err)
	}

	log.Printf("Processed client_created event for client ID %d", client.ID)
}

func (c *ClientConsumer) handleClientUpdated(ctx context.Context, client models.Client) {
	// 1. Обновляем в PostgreSQL
	if err := c.repo.UpdateClient(&client); err != nil {
		log.Printf("Failed to update client from Kafka: %v", err)
		return
	}

	// 2. Сериализуем клиента в JSON и обновляем кэш
	cacheKey := fmt.Sprintf("client:%d", client.ID)
	clientJSON, err := json.Marshal(client)
	if err != nil {
		log.Printf("Failed to marshal client to JSON: %v", err)
		return
	}

	if err := c.cache.SetToCache(ctx, cacheKey, string(clientJSON), 24*time.Hour); err != nil {
		log.Printf("Failed to update client in cache: %v", err)
	}

	log.Printf("Processed client_updated event for client ID %d", client.ID)
}

func (c *ClientConsumer) handleClientDeleted(ctx context.Context, clientID uint) {
	// Удаляем из Redis (используем пустую строку с TTL 0)
	cacheKey := fmt.Sprintf("client:%d", clientID)
	if err := c.cache.SetToCache(ctx, cacheKey, "", 0); err != nil {
		log.Printf("Failed to delete client from cache: %v", err)
	}

	log.Printf("Processed client_deleted event for client ID %d", clientID)
}
