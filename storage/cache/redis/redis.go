package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	goredis "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"time"
)

type CacheStore struct {
	client *goredis.ClusterClient
	logger *zap.Logger
}

const (
	otelName = "recommendationservice/internal/storage/cache/redis"
)

// NewCacheStore ...
func NewCacheStore(logger *zap.Logger, host, username, password string) *CacheStore {
	ctx := context.Background()
	conn := goredis.NewClusterClient(&goredis.ClusterOptions{
		Addrs:        []string{host},
		Username:     username,
		Password:     password,
		PoolSize:     10,
		MinIdleConns: 10,

		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		PoolTimeout:  4 * time.Second,

		MaxRetries:      10,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,

		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},

		ReadOnly:       false,
		RouteRandomly:  false,
		RouteByLatency: false,
	})

	c, err := conn.Ping(ctx).Result()

	if err != nil {
		logger.Error("failed to establish connection to redis cluster", zap.Error(err))
	}

	logger.Info(fmt.Sprintf("Redis Client sucessfully established connection with the AWS Elasticache Redis server with %v response returned from the server.", c))

	return &CacheStore{
		client: conn,
		logger: logger,
	}
}

func (c *CacheStore) Created() {
	// When an item is created, persist it into the cache
}

func (c *CacheStore) Deleted() {
	// When an item is deleted, from it from the cache store
}

func (c *CacheStore) Updated() {
	// When an item is updated, update it in the cache system
}

func newOTELSpan(ctx context.Context, name string) trace.Span {
	_, span := otel.Tracer(otelName).Start(ctx, name)

	return span
}
