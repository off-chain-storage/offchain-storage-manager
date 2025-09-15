package iface

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

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
	MongoDB
}
