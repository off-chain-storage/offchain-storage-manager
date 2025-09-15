package iface

import (
	"context"
	"time"

	"github.com/redis/rueidis/rueidislock"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ReadOnlyRedisDB interface {
	Get(key string) (string, error)
	Scan(cursor uint64, match string, count int64) ([]string, uint64, error)
}

type AccessRedisDB interface {
	ReadOnlyRedisDB

	Set(key string, value interface{}, expiration time.Duration) error
	Del(key string) error
	NewLocker() (rueidislock.Locker, error)
}

// RedisDB is a service interface for the Redis database
type RedisDB interface {
	AccessRedisDB

	// SetRedisConn()
}

type ReadOnlyMongoDB interface {
	FindOne(ctx context.Context, filter interface{}) (*mongo.SingleResult, error)
}

type AccessMongoDB interface {
	ReadOnlyMongoDB

	InsertOne(ctx context.Context, doc interface{}) (*mongo.InsertOneResult, error)
	UploadToGridFS(ctx context.Context, filename string, data []byte, metadata bson.M) error
	// UpdateOne(filter, update interface{}) (*mongo.UpdateResult, error)
	// DeleteOne(filter interface{}) (*mongo.DeleteResult, error)
}

// MongoDBDB is a service interface for the MongoDB database
type MongoDB interface {
	AccessMongoDB

	// SetMongoDBConn()
}

// DB is a service interface for the database
type DB interface {
	RedisDB
	MongoDB
}
