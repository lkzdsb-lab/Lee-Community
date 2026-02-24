package pkg

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
	topic  string
}

type KafkaConfig struct {
	Brokers []string
	Topic   string
}

func NewKafkaProducer(cfg KafkaConfig) (*KafkaProducer, error) {
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
	return &KafkaProducer{writer: w, topic: cfg.Topic}, nil
}

func (p *KafkaProducer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

func (p *KafkaProducer) Send(ctx context.Context, key string, value []byte) error {
	msg := kafka.Message{
		Key:   []byte(key),
		Value: value,
	}
	return p.writer.WriteMessages(ctx, msg)
}

func MakeKeyFromID(id uint64) string {
	return fmt.Sprintf("%d", id)
}
