package database

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"

	"backend/config"

	"github.com/redis/go-redis/v9"
)

func NewRedis(cfg *config.Redis) (*redis.Client, error) {
	options := &redis.Options{
		Addr:     net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port))),
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.Database,
	}
	if cfg.TLS {
		options.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: cfg.Host,
		}
	}

	client := redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping Redis: %w", err)
	}

	return client, nil
}
