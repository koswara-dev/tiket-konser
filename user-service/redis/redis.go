package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"user-service/config"

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
		log.Printf("Warning: Failed to connect to Redis at %s: %v. Caching functions might not be available.", addr, err)
		return nil
	}

	log.Printf("Redis connection successfully established at %s", addr)
	return Client
}

func BlacklistToken(token string, ttl time.Duration) error {
	if Client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	key := fmt.Sprintf("blacklist:token:%s", token)
	return Client.Set(Ctx, key, "true", ttl).Err()
}

func IsTokenBlacklisted(token string) (bool, error) {
	if Client == nil {
		return false, nil
	}
	key := fmt.Sprintf("blacklist:token:%s", token)
	exists, err := Client.Exists(Ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func SetUserForceLogout(userID uint) error {
	if Client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	key := fmt.Sprintf("force_logout:user:%d", userID)
	timestamp := time.Now().Unix()
	return Client.Set(Ctx, key, timestamp, 24*time.Hour).Err()
}

func GetUserForceLogoutTime(userID uint) (int64, error) {
	if Client == nil {
		return 0, nil
	}
	key := fmt.Sprintf("force_logout:user:%d", userID)
	val, err := Client.Get(Ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return val, nil
}
