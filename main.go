package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
)

func main() {
	// 1. Инициализация логгера
	logger := log.New(os.Stdout, "WELLNESS: ", log.LstdFlags|log.Lshortfile)

	// 2. Настройка HTTP-сервера
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// API Routes
	api := router.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}

	// 3. Запуск сервера
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Printf("Server is running on port %s", port)
	if err := router.Run(":" + port); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
}
