package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type RefreshTokenStore struct {
	client    *goredis.Client
	keyPrefix string
	timeout   time.Duration
}

func NewRefreshTokenStore(client *goredis.Client, keyPrefix string, timeout time.Duration) *RefreshTokenStore {
	return &RefreshTokenStore{
		client:    client,
		keyPrefix: keyPrefix,
		timeout:   timeout,
	}
}

func (s *RefreshTokenStore) Save(ctx context.Context, userID, tokenID string, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.client.Set(ctx, s.key(userID, tokenID), "1", ttl).Err()
}

func (s *RefreshTokenStore) Exists(ctx context.Context, userID, tokenID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	result, err := s.client.Exists(ctx, s.key(userID, tokenID)).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (s *RefreshTokenStore) Delete(ctx context.Context, userID, tokenID string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.client.Del(ctx, s.key(userID, tokenID)).Err()
}

func (s *RefreshTokenStore) key(userID, tokenID string) string {
	return fmt.Sprintf("%s:refresh:%s:%s", s.keyPrefix, userID, tokenID)
}
