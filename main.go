package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"
	"wellness-step-by-step/step-03/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	logger := log.New(os.Stdout, "WELLNESS: ", log.LstdFlags|log.Lshortfile)

	// Пытаемся подключиться к Redis с ретраями
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

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	api := router.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()

			// Простая проверка Redis
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Printf("Server is running on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
}
