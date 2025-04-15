package utils

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer interface {
	SendMessage(ctx context.Context, topic string, key, value []byte) error
	Close() error
}

type kafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer() (KafkaProducer, error) {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "localhost:9092"
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(broker),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 50 * time.Millisecond,
	}

	// Проверка подключения
	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Kafka: %w", err)
	}
	defer conn.Close()

	return &kafkaProducer{writer: writer}, nil
}

func (k *kafkaProducer) SendMessage(ctx context.Context, topic string, key, value []byte) error {
	return k.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	})
}

func (k *kafkaProducer) Close() error {
	return k.writer.Close()
}
