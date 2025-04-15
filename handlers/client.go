package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"wellness-step-by-step/step-04/models"
	"wellness-step-by-step/step-04/utils"

	"github.com/gin-gonic/gin"
)

type ClientHandler struct {
	repo  models.Repository
	kafka utils.KafkaProducer
}

func NewClientHandler(repo models.Repository, kafka utils.KafkaProducer) *ClientHandler {
	return &ClientHandler{
		repo:  repo,
		kafka: kafka,
	}
}

type ClientRequest struct {
	FullName string `json:"full_name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone" binding:"required,e164"`
}

type ClientResponse struct {
	ID       uint   `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

func (h *ClientHandler) CreateClient(c *gin.Context) {
	var req ClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client := &models.Client{
		FullName: req.FullName,
		Email:    req.Email,
		Phone:    req.Phone,
	}

	if err := h.repo.CreateClient(client); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.kafka != nil {
		go h.sendKafkaEvent("client_created", client)
	}

	c.JSON(http.StatusCreated, toClientResponse(client))
}

func (h *ClientHandler) GetClient(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client ID is required"})
		return
	}

	id, err := parseUint(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID format"})
		return
	}

	client, err := h.repo.GetClientByID(id)
	if err != nil {
		if err == models.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toClientResponse(client))
}

func (h *ClientHandler) UpdateClient(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client ID is required"})
		return
	}

	id, err := parseUint(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID format"})
		return
	}

	var req ClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := h.repo.GetClientByID(id)
	if err != nil {
		if err == models.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	client.FullName = req.FullName
	client.Email = req.Email
	client.Phone = req.Phone

	if err := h.repo.UpdateClient(client); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.kafka != nil {
		go h.sendKafkaEvent("client_updated", client)
	}

	c.JSON(http.StatusOK, toClientResponse(client))
}

func (h *ClientHandler) DeleteClient(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client ID is required"})
		return
	}

	id, err := parseUint(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID format"})
		return
	}

	if err := h.repo.DeleteClient(id); err != nil {
		if err == models.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.kafka != nil {
		go func(id uint) {
			event := map[string]interface{}{
				"event": "client_deleted",
				"id":    id,
			}
			h.sendRawKafkaEvent("client_events", event)
		}(id)
	}

	c.Status(http.StatusNoContent)
}

// Вспомогательные методы

func (h *ClientHandler) sendKafkaEvent(eventType string, client *models.Client) {
	event := map[string]interface{}{
		"event":    eventType,
		"id":       client.ID,
		"email":    client.Email,
		"fullName": client.FullName,
	}
	h.sendRawKafkaEvent("client_events", event)
}

func (h *ClientHandler) sendRawKafkaEvent(topic string, event interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jsonData, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal Kafka event: %v", err)
		return
	}

	if err := h.kafka.SendMessage(ctx, topic, nil, jsonData); err != nil {
		log.Printf("Failed to send Kafka message: %v", err)
	}
}

func toClientResponse(client *models.Client) ClientResponse {
	return ClientResponse{
		ID:       client.ID,
		FullName: client.FullName,
		Email:    client.Email,
		Phone:    client.Phone,
	}
}

func parseUint(s string) (uint, error) {
	var id uint
	_, err := fmt.Sscanf(s, "%d", &id)
	return id, err
}
