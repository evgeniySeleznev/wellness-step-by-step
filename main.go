package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wellness-step-by-step/step-06/consumer"
	"wellness-step-by-step/step-06/handlers"
	"wellness-step-by-step/step-06/models"
	"wellness-step-by-step/step-06/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	logger := log.New(os.Stdout, "WELLNESS: ", log.LstdFlags|log.Lshortfile)

	// 1. Инициализация Redis
	var redisClient utils.RedisClient
	var err error
	maxRetries := 5
	retryDelay := 3 * time.Second

	for i := 0; i < maxRetries; i++ {
		redisClient, err = utils.NewRedisClient()
		if err == nil {
			break
		}
		logger.Printf("Attempt %d: Failed to connect to Redis: %v", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	if err != nil {
		logger.Fatalf("Failed to initialize Redis after %d attempts: %v", maxRetries, err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Printf("Error closing Redis connection: %v", err)
		}
	}()

	// 2. Инициализация базы данных
	dbRepo, err := models.NewPostgresRepository()
	if err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := dbRepo.Close(); err != nil {
			logger.Printf("Error closing database connection: %v", err)
		}
	}()

	// 3. Инициализация Kafka
	kafkaProducer, err := utils.NewKafkaProducer()
	if err != nil {
		logger.Printf("WARNING: Kafka initialization failed: %v", err)
	} else {
		defer kafkaProducer.Close()
	}

	// 4. Инициализация Elasticsearch
	var esClient utils.ElasticsearchClient
	for i := 0; i < maxRetries; i++ {
		esClient, err = utils.NewElasticsearchClient()
		if err == nil {
			break
		}
		logger.Printf("Attempt %d: Failed to connect to Elasticsearch: %v", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	if err != nil {
		logger.Printf("WARNING: Elasticsearch initialization failed: %v", err)
	} else {
		defer func() {
			if err := esClient.Close(); err != nil {
				logger.Printf("Error closing Elasticsearch connection: %v", err)
			}
		}()
	}

	// 5. Инициализация обработчиков
	clientHandler := handlers.NewClientHandler(dbRepo, kafkaProducer, esClient)

	// 6. Инициализация Consumer
	clientConsumer := consumer.NewClientConsumer(dbRepo, redisClient, esClient)
	go clientConsumer.Start(context.Background())
	defer clientConsumer.Stop()

	// 7. Настройка маршрутов
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	api := router.Group("/api/v1")
	{
		api.POST("/clients", clientHandler.CreateClient)
		api.GET("/clients/:id", clientHandler.GetClient)
		api.PUT("/clients/:id", clientHandler.UpdateClient)
		api.DELETE("/clients/:id", clientHandler.DeleteClient)
		api.GET("/clients/search", clientHandler.SearchClients) // Новый endpoint для поиска

		api.GET("/health", func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()

			healthStatus := gin.H{
				"status": "ok",
				"details": gin.H{
					"redis":         "available",
					"postgres":      "available",
					"kafka":         "unknown",
					"elasticsearch": "unknown",
				},
			}

			// Проверка Redis
			if err := redisClient.SetToCache(ctx, "healthcheck", "ping", 10*time.Second); err != nil {
				healthStatus["status"] = "degraded"
				healthStatus["details"].(gin.H)["redis"] = "unavailable"
				healthStatus["error"] = err.Error()
			}

			// Проверка Elasticsearch
			if esClient != nil {
				healthStatus["details"].(gin.H)["elasticsearch"] = "available"
			} else {
				healthStatus["details"].(gin.H)["elasticsearch"] = "unavailable"
			}

			// Проверка Kafka
			if kafkaProducer != nil {
				healthStatus["details"].(gin.H)["kafka"] = "available"
			}

			if healthStatus["status"] == "ok" {
				c.JSON(http.StatusOK, healthStatus)
			} else {
				c.JSON(http.StatusServiceUnavailable, healthStatus)
			}
		})
	}

	// 8. Запуск сервера
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Настройка graceful shutdown
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Printf("Server is running on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	logger.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Println("Server exiting")
}
