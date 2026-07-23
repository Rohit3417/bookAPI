package cache

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(ctx context.Context) (*redis.Client, error) {

	opt, _ := redis.ParseURL(os.Getenv("REDIS_URL"))
	client := redis.NewClient(opt)

	err := client.Ping(ctx).Err()
	if err != nil {
		return nil, fmt.Errorf("redis connection failed :%w", err)
	}
	log.Println("Connected to redis")
	return client, nil
}

func BlacklistToken(ctx context.Context, client *redis.Client, token string, ttl time.Duration) error {
	key := "blacklist:" + token

	err := client.Set(ctx, key, "true", ttl).Err()
	if err != nil {
		return fmt.Errorf("Failed to blacklist token : %w", err)
	}

	return nil
}
