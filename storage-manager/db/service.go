package db

import (
	"context"
	"time"

	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/types"
	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/util"
	"github.com/redis/rueidis/rueidislock"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var log = util.NewLogger("db")

type DBService struct {
	ctx    context.Context
	cancel context.CancelFunc
	Redis  *RedisClient
	Mongo  *MongoDBClient
}

func NewDBService(cfg *types.Config) (*DBService, error) {
	// For upper service to cancel the context
	ctx, cancel := context.WithCancel(context.Background())

	// For init timeout (15 seconds) — only for readiness checks
	initCtx, initCancel := context.WithTimeout(ctx, 15*time.Second)
	defer initCancel()

	// Construct clients with the long-lived service ctx
	redisClient, err := NewRedisClient(ctx, cfg)
	if err != nil {
		cancel()
		return nil, err
	}

	mongoClient, err := NewMongoDBClient(ctx, cfg)
	if err != nil {
		_ = redisClient.Close()
		cancel()
		return nil, err
	}

	// Verify readiness within init timeout to fail fast if needed
	if err := redisClient.HealthCheckWith(initCtx); err != nil {
		_ = mongoClient.Close()
		_ = redisClient.Close()
		cancel()
		return nil, err
	}
	if err := mongoClient.HealthCheckWith(initCtx); err != nil {
		_ = mongoClient.Close()
		_ = redisClient.Close()
		cancel()
		return nil, err
	}

	// DB Service Instance
	d := &DBService{
		ctx:    ctx,
		cancel: cancel,
		Redis:  redisClient,
		Mongo:  mongoClient,
	}

	return d, nil
}

func (d *DBService) Start() {
	log.Info("Starting DB service")
	d.Redis.Start()
	d.Mongo.Start()
}

func (d *DBService) Stop() error {
	log.Info("Stopping DB service")
	return d.Close()
}

func (d *DBService) Close() error {
	log.Info("Closing DB service")
	d.cancel()
	d.Redis.Close()
	d.Mongo.Close()

	return nil
}

// Implement iface.RedisDB by delegating to the Redis client
func (d *DBService) Get(key string) (string, error) {
	return d.Redis.Get(key)
}

func (d *DBService) Scan(cursor uint64, match string, count int64) ([]string, uint64, error) {
	return d.Redis.Scan(cursor, match, count)
}

func (d *DBService) Set(key string, value interface{}, expiration time.Duration) error {
	return d.Redis.Set(key, value, expiration)
}

func (d *DBService) Del(key string) error {
	return d.Redis.Del(key)
}

func (d *DBService) NewLocker() (rueidislock.Locker, error) {
	return d.Redis.NewLocker()
}

// Implement iface.MongoDB by delegating to the Mongo client
func (d *DBService) InsertOne(ctx context.Context, doc interface{}) (*mongo.InsertOneResult, error) {
	return d.Mongo.InsertOne(ctx, doc)
}

func (d *DBService) FindOne(ctx context.Context, filter interface{}) (*mongo.SingleResult, error) {
	return d.Mongo.FindOne(ctx, filter)
}

func (d *DBService) UploadToGridFS(ctx context.Context, filename string, data []byte, metadata bson.M) error {
	return d.Mongo.UploadToGridFS(ctx, filename, data, metadata)
}
