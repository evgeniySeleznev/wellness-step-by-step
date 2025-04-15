package models

import (
	"errors"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
)

var ErrNotFound = errors.New("record not found")

type Repository interface {
	CreateClient(client *Client) error
	GetClientByID(id uint) (*Client, error) // Изменили тип id на uint
	UpdateClient(client *Client) error
	DeleteClient(id uint) error // Изменили тип id на uint
	Close() error
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository() (*PostgresRepository, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(&Client{}); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate database: %w", err)
	}

	return &PostgresRepository{db: db}, nil
}

func (r *PostgresRepository) CreateClient(client *Client) error {
	if err := r.db.Create(client).Error; err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetClientByID(id uint) (*Client, error) {
	var client Client
	if err := r.db.First(&client, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	return &client, nil
}

func (r *PostgresRepository) DeleteClient(id uint) error {
	result := r.db.Delete(&Client{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete client: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) UpdateClient(client *Client) error {
	result := r.db.Save(client)
	if result.Error != nil {
		return fmt.Errorf("failed to update client: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	return sqlDB.Close()
}
