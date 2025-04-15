package main

import (
	"log"
	"net/http"
	"os"
	"wellness-step-by-step/step-02/models"

	"github.com/gin-gonic/gin"
)

func main() {
	// Инициализация БД
	dbRepo, err := models.NewPostgresRepository()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbRepo.Close()

	router := gin.Default()

	// Простейший CRUD
	router.POST("/clients", func(c *gin.Context) {
		var client models.Client
		if err := c.ShouldBindJSON(&client); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := dbRepo.CreateClient(&client); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, client)
	})

	router.GET("/clients/:id", func(c *gin.Context) {
		id := c.Param("id")
		client, err := dbRepo.GetClientByID(id)
		if err != nil {
			if err == models.ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, client)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	router.Run(":" + port)
}
