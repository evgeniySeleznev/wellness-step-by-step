package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wellness-step-by-step/step-05/consumer"
	"wellness-step-by-step/step-05/handlers"
	"wellness-step-by-step/step-05/models"
	"wellness-step-by-step/step-05/utils"

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

	// 4. Инициализация обработчиков
	clientHandler := handlers.NewClientHandler(dbRepo, kafkaProducer)

	// 5. Настройка маршрутов
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	api := router.Group("/api/v1")
	{
		api.POST("/clients", clientHandler.CreateClient)
		api.GET("/clients/:id", clientHandler.GetClient)
		api.PUT("/clients/:id", clientHandler.UpdateClient)
		api.DELETE("/clients/:id", clientHandler.DeleteClient)

		api.GET("/health", func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()

			if err := redisClient.SetToCache(ctx, "healthcheck", "ping", 10*time.Second); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status":  "degraded",
					"details": gin.H{"redis": "unavailable"},
					"error":   err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"details": gin.H{"redis": "available"},
			})
		})
	}

	// 6. Запуск сервера
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Printf("Server is running on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
	// После инициализации всех компонентов
	clientConsumer := consumer.NewClientConsumer(dbRepo, redisClient)
	go clientConsumer.Start(context.Background())

	// Добавить в graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Остановка Consumer перед завершением
	clientConsumer.Stop()
}
