package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/event-bus/internal/kafka"
)

type Manager struct {
	producer      *kafka.Producer
	consumer      *kafka.Consumer
	subscriptions map[string]*Subscription
	mu            sync.RWMutex
	client        *http.Client
	dlqProducer   *kafka.Producer
}

type Subscription struct {
	ID                string
	UserID            string
	WebhookURL        string
	EventTypeFilter   string
	Active            bool
	SignatureSecret   string
	MaxRetries        int
	InitialDelayMS    int
	BackoffMultiplier float64
	CreatedAt         time.Time
}

type WebhookDelivery struct {
	DeliveryID    string
	SubscriptionID string
	EventID       string
	Attempt       int
	Status        string
	HTTPStatus    int
	DeliveredAt   time.Time
	LatencyMS     int
	ErrorMessage  string
}

func NewManager(producer *kafka.Producer, consumer *kafka.Consumer) *Manager {
	return &Manager{
		producer:      producer,
		consumer:      consumer,
		subscriptions: make(map[string]*Subscription),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (m *Manager) CreateSubscription(sub *Subscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.subscriptions[sub.ID]; exists {
		return fmt.Errorf("subscription %s already exists", sub.ID)
	}

	sub.CreatedAt = time.Now()
	m.subscriptions[sub.ID] = sub
	return nil
}

func (m *Manager) GetSubscription(id string) (*Subscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sub, exists := m.subscriptions[id]
	if !exists {
		return nil, fmt.Errorf("subscription %s not found", id)
	}

	return sub, nil
}

func (m *Manager) ListSubscriptions(userID string) []*Subscription {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Subscription
	for _, sub := range m.subscriptions {
		if sub.UserID == userID {
			result = append(result, sub)
		}
	}

	return result
}

func (m *Manager) UpdateSubscription(id string, updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sub, exists := m.subscriptions[id]
	if !exists {
		return fmt.Errorf("subscription %s not found", id)
	}

	if url, ok := updates["webhookURL"].(string); ok {
		sub.WebhookURL = url
	}
	if active, ok := updates["active"].(bool); ok {
		sub.Active = active
	}

	return nil
}

func (m *Manager) DeleteSubscription(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.subscriptions[id]; !exists {
		return fmt.Errorf("subscription %s not found", id)
	}

	delete(m.subscriptions, id)
	return nil
}

func (m *Manager) DeliverWebhook(ctx context.Context, sub *Subscription, event map[string]interface{}) (*WebhookDelivery, error) {
	if !sub.Active {
		return nil, fmt.Errorf("subscription is not active")
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	signature := m.generateSignature(payload, sub.SignatureSecret)

	for attempt := 1; attempt <= sub.MaxRetries; attempt++ {
		delivery := &WebhookDelivery{
			SubscriptionID: sub.ID,
			Attempt:        attempt,
			DeliveredAt:    time.Now(),
		}

		if eventID, ok := event["id"].(string); ok {
			delivery.EventID = eventID
		}

		start := time.Now()

		req, err := http.NewRequestWithContext(ctx, "POST", sub.WebhookURL, bytes.NewReader(payload))
		if err != nil {
			delivery.Status = "failed"
			delivery.ErrorMessage = err.Error()
			m.recordDelivery(delivery)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Webhook-Signature", signature)
		req.Header.Set("X-Attempt", fmt.Sprintf("%d", attempt))

		resp, err := m.client.Do(req)
		delivery.LatencyMS = int(time.Since(start).Milliseconds())

		if err != nil {
			delivery.Status = "failed"
			delivery.ErrorMessage = err.Error()
			m.recordDelivery(delivery)

			if attempt < sub.MaxRetries {
				delay := m.calculateBackoff(attempt, sub.InitialDelayMS, sub.BackoffMultiplier)
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}
			continue
		}

		defer resp.Body.Close()
		delivery.HTTPStatus = resp.StatusCode

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			delivery.Status = "delivered"
			m.recordDelivery(delivery)
			return delivery, nil
		}

		delivery.Status = "failed"
		body, _ := io.ReadAll(resp.Body)
		delivery.ErrorMessage = string(body)
		m.recordDelivery(delivery)

		if attempt < sub.MaxRetries {
			delay := m.calculateBackoff(attempt, sub.InitialDelayMS, sub.BackoffMultiplier)
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
	}

	return nil, fmt.Errorf("webhook delivery failed after %d attempts", sub.MaxRetries)
}

func (m *Manager) generateSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

func (m *Manager) calculateBackoff(attempt int, initialDelayMS int, multiplier float64) int {
	delay := float64(initialDelayMS)
	for i := 1; i < attempt; i++ {
		delay *= multiplier
	}
	return int(delay)
}

func (m *Manager) recordDelivery(delivery *WebhookDelivery) {
	// TODO: Store delivery history in database or DLQ
}

func (m *Manager) ProcessEvent(ctx context.Context, event map[string]interface{}) error {
	m.mu.RLock()
	subscriptions := make([]*Subscription, 0, len(m.subscriptions))
	for _, sub := range m.subscriptions {
		subscriptions = append(subscriptions, sub)
	}
	m.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(subscriptions))

	eventType, _ := event["type"].(string)

	for _, sub := range subscriptions {
		if !m.matchesFilter(eventType, sub.EventTypeFilter) {
			continue
		}

		wg.Add(1)
		go func(s *Subscription) {
			defer wg.Done()
			_, err := m.DeliverWebhook(ctx, s, event)
			if err != nil {
				errChan <- err
			}
		}(sub)
	}

	wg.Wait()
	close(errChan)

	return nil
}

func (m *Manager) matchesFilter(eventType, filter string) bool {
	if filter == "*" || filter == "" {
		return true
	}
	if filter == eventType {
		return true
	}
	return false
}
