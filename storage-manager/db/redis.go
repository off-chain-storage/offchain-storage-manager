package db

import (
	"context"
	"fmt"
	"time"

	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/types"
	"github.com/redis/rueidis"
	"github.com/redis/rueidis/rueidislock"
)

type RedisClient struct {
	client rueidis.Client
	ctx    context.Context
	cancel context.CancelFunc
	cfg    *types.Config
}

func NewRedisClient(ctx context.Context, cfg *types.Config) (*RedisClient, error) {
	ctx, cancel := context.WithCancel(ctx)

	redisAddr := fmt.Sprintf("%s:%s", cfg.DB.Redis.Host, cfg.DB.Redis.Port)
	client, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{redisAddr},
		Password:    cfg.DB.Redis.Pass,
		SelectDB:    cfg.DB.Redis.Name,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	redisClient := &RedisClient{
		client: client,
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg,
	}

	return redisClient, nil
}

func (r *RedisClient) Start() {
	log.Info("Start Redis client")
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-r.ctx.Done():
				log.Info("Redis health check loop exiting due to context cancellation")
				return
			case <-ticker.C:
				_ = r.HealthCheck()
			}
		}
	}()
}

func (r *RedisClient) Stop() error {
	log.Info("Stop Redis client")
	return r.Close()
}

func (r *RedisClient) HealthCheck() error {
	cmd := r.client.B().Ping().Build()
	res := r.client.Do(r.ctx, cmd)
	err := res.Error()
	if err != nil {
		log.WithError(err).Warn("Redis health check failed")
	} else {
		log.Debug("Redis health check passed")
	}
	return err
}

// HealthCheckWith runs a ping using the provided context.
func (r *RedisClient) HealthCheckWith(ctx context.Context) error {
	cmd := r.client.B().Ping().Build()
	res := r.client.Do(ctx, cmd)
	err := res.Error()
	if err != nil {
		log.WithError(err).Warn("Redis health check failed")
	} else {
		log.Debug("Redis health check passed")
	}
	return err
}

func (r *RedisClient) Close() error {
	r.cancel()
	r.client.Close()
	return nil
}

func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	var cmd rueidis.Completed
	if expiration > 0 {
		cmd = r.client.B().Set().Key(key).Value(fmt.Sprint(value)).Ex(expiration).Build()
	} else {
		cmd = r.client.B().Set().Key(key).Value(fmt.Sprint(value)).Build()
	}
	return r.client.Do(r.ctx, cmd).Error()
}

func (r *RedisClient) Get(key string) (string, error) {
	cmd := r.client.B().Get().Key(key).Build()
	return r.client.Do(r.ctx, cmd).ToString()
}

func (r *RedisClient) Del(key string) error {
	cmd := r.client.B().Del().Key(key).Build()
	return r.client.Do(r.ctx, cmd).Error()
}

func (r *RedisClient) Scan(cursor uint64, match string, count int64) ([]string, uint64, error) {
	cmd := r.client.B().Scan().Cursor(cursor).Match(match).Count(count).Build()
	res := r.client.Do(r.ctx, cmd)
	vals, err := res.ToArray()
	if err != nil || len(vals) != 2 {
		return nil, 0, fmt.Errorf("unexpected scan response: %v", err)
	}
	nextCursor, err := vals[0].AsUint64()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse cursor: %v", err)
	}
	keys, err := vals[1].AsStrSlice()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse keys: %v", err)
	}
	return keys, nextCursor, nil
}

// NewLocker creates a new distributed locker using rueidislock with existing Redis client.
func (r *RedisClient) NewLocker() (rueidislock.Locker, error) {
	return rueidislock.NewLocker(rueidislock.LockerOption{
		KeyMajority: 1,
		ClientBuilder: func(_ rueidis.ClientOption) (rueidis.Client, error) {
			return r.client, nil
		},
	})
}
