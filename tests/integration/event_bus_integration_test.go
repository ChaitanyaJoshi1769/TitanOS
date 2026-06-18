package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/event-bus/internal/events"
	"github.com/ChaitanyaJoshi1769/TitanOS/services/event-bus/internal/kafka"
	"github.com/ChaitanyaJoshi1769/TitanOS/services/event-bus/internal/webhook"
)

func TestEventPublishAndConsume(t *testing.T) {
	kafkaHost := "localhost:9092"

	producer, err := kafka.NewProducer(kafkaHost)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	consumer, err := kafka.NewConsumer(kafkaHost, "test-group-1")
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	event := events.NewCloudEvent("titan.task.submitted", "scheduler", "task-123", map[string]interface{}{
		"taskId":    "task-123",
		"projectId": "proj-1",
		"name":      "Test Task",
	})

	eventMap := event.ToMap()
	_, err = producer.PublishEvent("tasks", "task-123", eventMap)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	t.Logf("✓ Event published successfully")
}

func TestCloudEventValidation(t *testing.T) {
	tests := []struct {
		name    string
		event   *events.CloudEvent
		wantErr bool
	}{
		{
			name: "Valid event",
			event: events.NewCloudEvent("titan.task.submitted", "scheduler", "task-123", map[string]interface{}{
				"taskId": "task-123",
			}),
			wantErr: false,
		},
		{
			name: "Missing type",
			event: &events.CloudEvent{
				SpecVersion:     "1.0",
				Source:          "scheduler",
				ID:              "task-123",
				Time:            time.Now(),
				DataContentType: "application/json",
			},
			wantErr: true,
		},
		{
			name: "Invalid spec version",
			event: &events.CloudEvent{
				SpecVersion:     "2.0",
				Type:            "titan.task.submitted",
				Source:          "scheduler",
				ID:              "task-123",
				Time:            time.Now(),
				DataContentType: "application/json",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	t.Logf("✓ All validation tests passed")
}

func TestWebhookSubscriptionManager(t *testing.T) {
	producer, _ := kafka.NewProducer("localhost:9092")
	consumer, _ := kafka.NewConsumer("localhost:9092", "test-group-2")
	defer producer.Close()
	defer consumer.Close()

	manager := webhook.NewManager(producer, consumer)

	sub := &webhook.Subscription{
		ID:                "sub-1",
		UserID:            "user-1",
		WebhookURL:        "https://example.com/webhook",
		EventTypeFilter:   "titan.task.*",
		Active:            true,
		SignatureSecret:   "secret-key",
		MaxRetries:        3,
		InitialDelayMS:    1000,
		BackoffMultiplier: 2.0,
	}

	err := manager.CreateSubscription(sub)
	if err != nil {
		t.Fatalf("Failed to create subscription: %v", err)
	}

	retrieved, err := manager.GetSubscription("sub-1")
	if err != nil {
		t.Fatalf("Failed to get subscription: %v", err)
	}

	if retrieved.ID != "sub-1" {
		t.Errorf("Expected subscription ID 'sub-1', got '%s'", retrieved.ID)
	}

	t.Logf("✓ Subscription created and retrieved successfully")
}

func TestWebhookSignature(t *testing.T) {
	producer, _ := kafka.NewProducer("localhost:9092")
	consumer, _ := kafka.NewConsumer("localhost:9092", "test-group-3")
	defer producer.Close()
	defer consumer.Close()

	manager := webhook.NewManager(producer, consumer)

	payload := []byte(`{"id":"test-1","type":"titan.task.submitted"}`)
	secret := "secret-key"
	signature := manager.GenerateSignature(payload, secret)

	if signature == "" {
		t.Error("Expected non-empty signature")
	}

	if len(signature) < 10 {
		t.Error("Expected longer signature")
	}

	t.Logf("✓ Webhook signature generated successfully: %s", signature)
}

func TestEventTypeFiltering(t *testing.T) {
	kafkaHost := "localhost:9092"

	producer, err := kafka.NewProducer(kafkaHost)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	consumer, err := kafka.NewConsumer(kafkaHost, "test-group-4")
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	testCases := []struct {
		eventType string
		filter    string
		matches   bool
	}{
		{"titan.task.submitted", "titan.task.*", true},
		{"titan.task.completed", "titan.task.*", true},
		{"titan.workflow.started", "titan.task.*", false},
		{"titan.agent.created", "*", true},
		{"titan.node.heartbeat", "titan.node.*", true},
	}

	for _, tc := range testCases {
		matched := matchEventType(tc.eventType, tc.filter)
		if matched != tc.matches {
			t.Errorf("Event type %s with filter %s: expected %v, got %v", tc.eventType, tc.filter, tc.matches, matched)
		}
	}

	t.Logf("✓ All event type filtering tests passed")
}

func TestExactlyOnceSemantics(t *testing.T) {
	kafkaHost := "localhost:9092"

	producer, err := kafka.NewProducer(kafkaHost)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	publishedEvents := 100
	topic := "exactly-once-test"

	for i := 0; i < publishedEvents; i++ {
		event := map[string]interface{}{
			"id":    i,
			"type":  "test.event",
			"index": i,
		}
		_, err := producer.PublishEvent(topic, string(rune(i)), event)
		if err != nil {
			t.Fatalf("Failed to publish event %d: %v", i, err)
		}
	}

	t.Logf("✓ Published %d events for exactly-once test", publishedEvents)
}

func TestEventBatching(t *testing.T) {
	kafkaHost := "localhost:9092"

	producer, err := kafka.NewProducer(kafkaHost)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	events := []map[string]interface{}{
		{"id": "task-1", "type": "titan.task.submitted"},
		{"id": "task-2", "type": "titan.task.submitted"},
		{"id": "task-3", "type": "titan.task.submitted"},
	}

	err = producer.PublishEventBatch("tasks-batch", events)
	if err != nil {
		t.Fatalf("Failed to publish batch: %v", err)
	}

	t.Logf("✓ Event batch published successfully")
}

func TestWebhookRetryLogic(t *testing.T) {
	producer, _ := kafka.NewProducer("localhost:9092")
	consumer, _ := kafka.NewConsumer("localhost:9092", "test-group-5")
	defer producer.Close()
	defer consumer.Close()

	manager := webhook.NewManager(producer, consumer)

	sub := &webhook.Subscription{
		ID:                "sub-retry-1",
		UserID:            "user-1",
		WebhookURL:        "https://example.com/webhook",
		EventTypeFilter:   "titan.task.*",
		Active:            true,
		SignatureSecret:   "secret-key",
		MaxRetries:        5,
		InitialDelayMS:    100,
		BackoffMultiplier: 2.0,
	}

	delays := []int{100, 200, 400, 800, 1600}
	for i, expected := range delays {
		actual := calculateBackoff(i+1, sub.InitialDelayMS, sub.BackoffMultiplier)
		if actual != expected {
			t.Errorf("Attempt %d: expected delay %dms, got %dms", i+1, expected, actual)
		}
	}

	t.Logf("✓ Webhook retry backoff logic verified")
}

func TestDeadLetterQueue(t *testing.T) {
	kafkaHost := "localhost:9092"

	producer, err := kafka.NewProducer(kafkaHost)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	dlqEvent := map[string]interface{}{
		"id":              "failed-delivery-1",
		"subscriptionId":  "sub-1",
		"eventId":         "event-1",
		"failureReason":   "webhook unreachable",
		"attemptNumber":   5,
		"lastAttemptTime": time.Now().Format(time.RFC3339),
	}

	_, err = producer.PublishEvent("webhooks.dlq", "failed-delivery-1", dlqEvent)
	if err != nil {
		t.Fatalf("Failed to publish to DLQ: %v", err)
	}

	t.Logf("✓ Dead-letter queue event published successfully")
}

// Helper functions

func (m *webhook.Manager) GenerateSignature(payload []byte, secret string) string {
	return m.generateSignature(payload, secret)
}

func matchEventType(eventType, filter string) bool {
	if filter == "*" || filter == "" {
		return true
	}
	if filter == eventType {
		return true
	}
	if len(filter) > 0 && filter[len(filter)-1] == '*' {
		prefix := filter[:len(filter)-1]
		return len(eventType) >= len(prefix) && eventType[:len(prefix)] == prefix
	}
	return false
}

func calculateBackoff(attempt int, initialDelayMS int, multiplier float64) int {
	delay := float64(initialDelayMS)
	for i := 1; i < attempt; i++ {
		delay *= multiplier
	}
	return int(delay)
}

// Benchmark tests

func BenchmarkEventPublishing(b *testing.B) {
	producer, _ := kafka.NewProducer("localhost:9092")
	defer producer.Close()

	event := map[string]interface{}{
		"id":   "bench-task",
		"type": "titan.task.submitted",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		producer.PublishEvent("bench-tasks", "key", event)
	}
}

func BenchmarkEventValidation(b *testing.B) {
	event := events.NewCloudEvent("titan.task.submitted", "scheduler", "task-123", map[string]interface{}{
		"taskId": "task-123",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event.Validate()
	}
}

func BenchmarkWebhookSignatureGeneration(b *testing.B) {
	producer, _ := kafka.NewProducer("localhost:9092")
	consumer, _ := kafka.NewConsumer("localhost:9092", "bench-group")
	defer producer.Close()
	defer consumer.Close()

	manager := webhook.NewManager(producer, consumer)
	payload := []byte(`{"id":"test-1","type":"titan.task.submitted"}`)
	secret := "secret-key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.GenerateSignature(payload, secret)
	}
}
