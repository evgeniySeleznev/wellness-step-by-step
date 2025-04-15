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
	GetClientByID(id string) (*Client, error)
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
	return r.db.Create(client).Error
}

func (r *PostgresRepository) GetClientByID(id string) (*Client, error) {
	var client Client
	if err := r.db.First(&client, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &client, nil
}

func (r *PostgresRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
