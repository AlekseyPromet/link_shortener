package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBNumber int    `json:"db_number"`
}

func LoadRedisConfig(filePath string) (*RedisConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %w", err)
	}
	defer file.Close()

	var config RedisConfig
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	slog.Info("Loaded config from file")

	return &config, nil
}

func NewRedisConnection(ctx context.Context, cfg *RedisConfig) (*redis.Client, error) {

	options, err := redis.ParseURL(fmt.Sprintf("redis://%s:%s@%s%d/%d",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBNumber))

	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	connect := redis.NewClient(options)

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	state := connect.Ping(ctx)

	if state.Err() != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", state.Err())
	}

	slog.Info("Connected to Redis")

	return connect, nil
}
