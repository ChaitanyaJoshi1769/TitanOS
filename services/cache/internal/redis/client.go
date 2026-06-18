package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	client *redis.Client
	mu     sync.RWMutex
}

type CacheEntry struct {
	Key        string
	Value      interface{}
	ExpiresAt  time.Time
	TTL        time.Duration
	AccessCount int64
	LastAccessed time.Time
}

func NewClient(ctx context.Context, addr string) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
		PoolTimeout:  4 * time.Second,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{
		client: rdb,
	}, nil
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *Client) Get(ctx context.Context, key string, dest interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("key not found: %s", key)
	}
	if err != nil {
		return fmt.Errorf("failed to get key %s: %w", key, err)
	}

	return json.Unmarshal([]byte(val), dest)
}

func (c *Client) Delete(ctx context.Context, keys ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.client.Del(ctx, keys...).Err()
}

func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}

	return exists > 0, nil
}

func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.client.Expire(ctx, key, ttl).Err()
}

func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ttl, err := c.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL: %w", err)
	}

	return ttl, nil
}

func (c *Client) Increment(ctx context.Context, key string, increment int64) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	result, err := c.client.IncrBy(ctx, key, increment).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment key %s: %w", key, err)
	}

	return result, nil
}

func (c *Client) Decrement(ctx context.Context, key string, decrement int64) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	result, err := c.client.DecrBy(ctx, key, decrement).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to decrement key %s: %w", key, err)
	}

	return result, nil
}

func (c *Client) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal value: %w", err)
	}

	ok, err := c.client.SetNX(ctx, key, data, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to set key: %w", err)
	}

	return ok, nil
}

func (c *Client) GetSet(ctx context.Context, key string, value interface{}) (interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal value: %w", err)
	}

	result, err := c.client.GetSet(ctx, key, data).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get and set key: %w", err)
	}

	return result, nil
}

func (c *Client) MSet(ctx context.Context, values map[string]interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	args := make([]interface{}, 0, len(values)*2)
	for key, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
		}
		args = append(args, key, data)
	}

	return c.client.MSet(ctx, args...).Err()
}

func (c *Client) MGet(ctx context.Context, keys ...string) (map[string]interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	vals, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get multiple keys: %w", err)
	}

	result := make(map[string]interface{})
	for i, key := range keys {
		if vals[i] != nil {
			result[key] = vals[i]
		}
	}

	return result, nil
}

// Set Operations

func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.client.SAdd(ctx, key, members...).Err()
}

func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	members, err := c.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get set members: %w", err)
	}

	return members, nil
}

func (c *Client) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ok, err := c.client.SIsMember(ctx, key, member).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check set membership: %w", err)
	}

	return ok, nil
}

// Sorted Set Operations

func (c *Client) ZAdd(ctx context.Context, key string, score float64, member interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.client.ZAdd(ctx, key, redis.Z{Score: score, Member: member}).Err()
}

func (c *Client) ZRange(ctx context.Context, key string, start int64, stop int64) ([]interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	members, err := c.client.ZRange(ctx, key, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get sorted set range: %w", err)
	}

	result := make([]interface{}, len(members))
	for i, m := range members {
		result[i] = m
	}

	return result, nil
}

// Pub/Sub

func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return c.client.Publish(ctx, channel, data).Err()
}

func (c *Client) Subscribe(ctx context.Context, channels ...string) (*redis.PubSub, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.client.Subscribe(ctx, channels...), nil
}

// Info and Stats

func (c *Client) Info(ctx context.Context) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	info, err := c.client.Info(ctx).Result()
	if err != nil {
		return "", fmt.Errorf("failed to get info: %w", err)
	}

	return info, nil
}

func (c *Client) FlushDB(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.client.FlushDB(ctx).Err()
}

func (c *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}

	return keys, nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.client.Close()
}
