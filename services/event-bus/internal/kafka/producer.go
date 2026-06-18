package kafka

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
	mu     sync.RWMutex
}

type ProducedMessage struct {
	Partition int
	Offset    int64
}

func NewProducer(brokers ...string) (*Producer, error) {
	w := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Compression:            kafka.Snappy,
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireAll,
	}

	p := &Producer{
		writer: w,
	}

	return p, nil
}

func (p *Producer) PublishEvent(topic string, key string, event interface{}) (*ProducedMessage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	messages := []kafka.Message{
		{
			Key:   []byte(key),
			Value: eventBytes,
		},
	}

	err = p.writer.WriteMessages(nil, messages...)
	if err != nil {
		return nil, fmt.Errorf("failed to publish event to topic %s: %w", topic, err)
	}

	return &ProducedMessage{
		Partition: 0,
		Offset:    0,
	}, nil
}

func (p *Producer) PublishEventBatch(topic string, events []map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	messages := make([]kafka.Message, 0, len(events))
	for _, event := range events {
		eventBytes, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}

		key := ""
		if id, ok := event["id"]; ok {
			key = fmt.Sprint(id)
		}

		messages = append(messages, kafka.Message{
			Key:   []byte(key),
			Value: eventBytes,
		})
	}

	err := p.writer.WriteMessages(nil, messages...)
	if err != nil {
		return fmt.Errorf("failed to publish batch to topic %s: %w", topic, err)
	}

	return nil
}

func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.writer.Close()
}
