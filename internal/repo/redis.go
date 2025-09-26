package repo

import (
	"context"
	"fmt"
	"log"
	"project/config"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(config *config.Config) *redisClient {
	Rdb := redis.NewClient(&redis.Options{
		Addr:     config.Redis.RedisAddr,
		Password: config.Redis.Password,
		DB:       0,
	})

	_, err := Rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}
	return &redisClient{
		rdb: Rdb,
	}
}

type redisClient struct {
	rdb *redis.Client
}

func (s *redisClient) SaveToken(ctx context.Context, userID int64, token string, ttl time.Duration) error {

	key := buildTokenKey(userID)
	return s.rdb.Set(ctx, key, token, ttl).Err()
}
func (s *redisClient) DeleteToken(ctx context.Context, userID int64) error {
	key := buildTokenKey(userID)
	return s.rdb.Del(ctx, key).Err()
}
func buildTokenKey(userID int64) string {
	return "auth:token:" + fmt.Sprint(userID)
}
func (s *redisClient) GetToken(ctx context.Context, userID int64) (string, error) {
	key := buildTokenKey(userID)
	val, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}
