package main

import (
	"context"
	"fmt"
	"github.com/getsentry/sentry-go"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"
	"wellness-step-by-step/step-07/consumer"
	"wellness-step-by-step/step-07/handlers"
	"wellness-step-by-step/step-07/middleware"
	"wellness-step-by-step/step-07/models"
	"wellness-step-by-step/step-07/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	logger := log.New(os.Stdout, "WELLNESS: ", log.LstdFlags|log.Lshortfile)

	// Инициализация Sentry
	// После загрузки .env
	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			Debug:            true,
			Environment:      os.Getenv("APP_ENV"),
			Release:          "wellness@" + os.Getenv("APP_VERSION"),
			TracesSampleRate: 1.0, // 100% для теста
			AttachStacktrace: true,
		})
		if err != nil {
			log.Fatalf("Sentry init failed: %v", err)
		}
		log.Println("Sentry initialized with DSN:", dsn)

		// Flush перед выходом
		defer sentry.Flush(2 * time.Second)
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:   os.Getenv("SENTRY_DSN"),
		Debug: true, // Включите для тестов
	}); err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}

	// Обработка паник
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("application panic: %v", r)
			utils.CaptureError(err, map[string]interface{}{
				"stack": string(debug.Stack()),
			})
			logger.Fatalf("Application panic: %v", r)
		}
	}()

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
	router.Use(middleware.SentryMiddleware())
	router.Use(gin.Logger(), gin.Recovery())

	// Добавляем тестовый маршрут
	router.GET("/test-sentry", func(c *gin.Context) {
		// Вариант 1: Через Sentry напрямую
		sentry.CaptureException(fmt.Errorf("тестовая ошибка Sentry %s", time.Now()))

		// Вариант 2: Паника (должна перехватываться)
		panic("тестовая паника " + time.Now().Format(time.RFC3339))
	})

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
