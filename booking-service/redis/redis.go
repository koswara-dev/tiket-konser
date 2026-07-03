package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"booking-service/config"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client
var Ctx = context.Background()

func InitRedis(cfg *config.Config) *redis.Client {
	addr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)
	Client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(Ctx, 2*time.Second)
	defer cancel()

	_, err := Client.Ping(ctx).Result()
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis at %s: %v. Lock/Cache functions might not be available.", addr, err)
		return nil
	}

	log.Printf("Redis connection successfully established at %s", addr)
	return Client
}

func AcquireLock(key string, expiration time.Duration) bool {
	if Client == nil {
		log.Println("Warning: Redis client not initialized. Lock request bypassed.")
		return true
	}

	lockKey := fmt.Sprintf("lock:%s", key)
	val, err := Client.SetNX(Ctx, lockKey, "locked", expiration).Result()
	if err != nil {
		log.Printf("Error acquiring Redis lock for %s: %v", lockKey, err)
		return false
	}
	return val
}

func ReleaseLock(key string) {
	if Client == nil {
		return
	}
	lockKey := fmt.Sprintf("lock:%s", key)
	Client.Del(Ctx, lockKey)
}
