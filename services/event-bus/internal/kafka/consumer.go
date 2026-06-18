package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
	mu     sync.RWMutex
}

type ConsumedMessage struct {
	Topic     string
	Partition int
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   map[string]string
}

func NewConsumer(broker string, groupID string) (*Consumer, error) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:         []string{broker},
		GroupID:         groupID,
		StartOffset:     kafka.LastOffset,
		CommitInterval:  1000,
		SessionTimeout:  10000,
		RebalanceTimeout: 60000,
	})

	c := &Consumer{
		reader: r,
	}

	return c, nil
}

func (c *Consumer) SubscribeToTopics(ctx context.Context, topics ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.reader.JoinGroup(topics)
}

func (c *Consumer) ReadMessage(ctx context.Context) (*ConsumedMessage, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	msg, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	headers := make(map[string]string)
	for _, h := range msg.Headers {
		headers[h.Key] = string(h.Value)
	}

	return &ConsumedMessage{
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
		Key:       msg.Key,
		Value:     msg.Value,
		Headers:   headers,
	}, nil
}

func (c *Consumer) ReadMessageBatch(ctx context.Context, count int) ([]*ConsumedMessage, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	messages := make([]*ConsumedMessage, 0, count)

	for i := 0; i < count; i++ {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			return messages, fmt.Errorf("failed to read message %d: %w", i, err)
		}

		headers := make(map[string]string)
		for _, h := range msg.Headers {
			headers[h.Key] = string(h.Value)
		}

		messages = append(messages, &ConsumedMessage{
			Topic:     msg.Topic,
			Partition: msg.Partition,
			Offset:    msg.Offset,
			Key:       msg.Key,
			Value:     msg.Value,
			Headers:   headers,
		})
	}

	return messages, nil
}

func (c *Consumer) CommitMessages(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.reader.CommitMessages(ctx)
}

func (c *Consumer) ParseMessageAsJSON(msg *ConsumedMessage) (map[string]interface{}, error) {
	var data map[string]interface{}
	err := json.Unmarshal(msg.Value, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse message as JSON: %w", err)
	}
	return data, nil
}

func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reader.Close()
}
