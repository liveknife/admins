package database

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	URL      string
	Host     string
	Port     string
	Password string
	DB       int
	PoolSize int
}

func DefaultRedisConfig() RedisConfig {
	cfg := RedisConfig{
		URL:      os.Getenv("REDIS_URL"),
		Host:     envOr("REDIS_HOST", "localhost"),
		Port:     envOr("REDIS_PORT", "6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
		PoolSize: 10,
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if n, _ := strconv.Atoi(v); n >= 0 {
			cfg.DB = n
		}
	}
	if v := os.Getenv("REDIS_POOL_SIZE"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 {
			cfg.PoolSize = n
		}
	}
	return cfg
}

var RedisClient *redis.Client

func ConnectRedis() (*redis.Client, error) {
	cfg := DefaultRedisConfig()
	var client *redis.Client
	if cfg.URL != "" {
		opt, err := redis.ParseURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("parse REDIS_URL failed: %w", err)
		}
		client = redis.NewClient(opt)
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     cfg.Host + ":" + cfg.Port,
			Password: cfg.Password,
			DB:       cfg.DB,
			PoolSize: cfg.PoolSize,
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	fmt.Printf("[REDIS] connected successfully (%s:%s/%d)\n", cfg.Host, cfg.Port, cfg.DB)
	RedisClient = client
	return client, nil
}
