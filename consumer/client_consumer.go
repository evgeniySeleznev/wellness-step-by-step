package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
	"wellness-step-by-step/step-06/models"
	"wellness-step-by-step/step-06/utils"

	"github.com/segmentio/kafka-go"
)

type ClientEvent struct {
	Event string        `json:"event"`
	Data  models.Client `json:"data"`
}

// Добавляем поле es в структуру ClientConsumer
type ClientConsumer struct {
	repo     models.Repository
	cache    utils.RedisClient
	es       utils.ElasticsearchClient
	reader   *kafka.Reader
	shutdown chan struct{}
}

// Обновляем конструктор
func NewClientConsumer(repo models.Repository, cache utils.RedisClient, es utils.ElasticsearchClient) *ClientConsumer {
	return &ClientConsumer{
		repo:  repo,
		cache: cache,
		es:    es,
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

// Обновляем обработчики событий
func (c *ClientConsumer) handleClientCreated(ctx context.Context, client models.Client) {
	// 1. Сохраняем в PostgreSQL
	existing, err := c.repo.GetClientByID(client.ID)
	if err == nil && existing != nil {
		return // Клиент уже существует
	}

	if err := c.repo.CreateClient(&client); err != nil {
		log.Printf("Failed to create client from Kafka: %v", err)
		return
	}

	// 2. Сохраняем в Redis
	cacheKey := fmt.Sprintf("client:%d", client.ID)
	clientJSON, err := json.Marshal(client)
	if err != nil {
		log.Printf("Failed to marshal client to JSON: %v", err)
		return
	}

	if err := c.cache.SetToCache(ctx, cacheKey, string(clientJSON), 24*time.Hour); err != nil {
		log.Printf("Failed to cache client: %v", err)
	}

	// 3. Индексируем в Elasticsearch
	if c.es != nil {
		if err := c.es.IndexClient(ctx, "clients", fmt.Sprintf("%d", client.ID), client); err != nil {
			log.Printf("Failed to index client in Elasticsearch: %v", err)
		}
	}

	log.Printf("Processed client_created event for client ID %d", client.ID)
}

func (c *ClientConsumer) handleClientUpdated(ctx context.Context, client models.Client) {
	// 1. Обновляем в PostgreSQL
	if err := c.repo.UpdateClient(&client); err != nil {
		log.Printf("Failed to update client from Kafka: %v", err)
		return
	}

	// 2. Обновляем в Redis
	cacheKey := fmt.Sprintf("client:%d", client.ID)
	clientJSON, err := json.Marshal(client)
	if err != nil {
		log.Printf("Failed to marshal client to JSON: %v", err)
		return
	}

	if err := c.cache.SetToCache(ctx, cacheKey, string(clientJSON), 24*time.Hour); err != nil {
		log.Printf("Failed to update client in cache: %v", err)
	}

	// 3. Обновляем в Elasticsearch
	if c.es != nil {
		if err := c.es.IndexClient(ctx, "clients", fmt.Sprintf("%d", client.ID), client); err != nil {
			log.Printf("Failed to update client in Elasticsearch: %v", err)
		}
	}

	log.Printf("Processed client_updated event for client ID %d", client.ID)
}

func (c *ClientConsumer) handleClientDeleted(ctx context.Context, clientID uint) {
	// Удаляем из Redis
	cacheKey := fmt.Sprintf("client:%d", clientID)
	if err := c.cache.SetToCache(ctx, cacheKey, "", 0); err != nil {
		log.Printf("Failed to delete client from cache: %v", err)
	}

	// Удаляем из Elasticsearch
	if c.es != nil {
		if err := c.es.DeleteClient(ctx, "clients", fmt.Sprintf("%d", clientID)); err != nil {
			log.Printf("Failed to delete client from Elasticsearch: %v", err)
		}
	}

	log.Printf("Processed client_deleted event for client ID %d", clientID)
}
