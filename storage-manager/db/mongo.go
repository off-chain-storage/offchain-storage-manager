package db

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/types"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type MongoDBClient struct {
	db     *mongo.Database
	ctx    context.Context
	cancel context.CancelFunc
	cfg    *types.Config
}

func NewMongoDBClient(ctx context.Context, cfg *types.Config) (*MongoDBClient, error) {
	ctx, cancel := context.WithCancel(ctx)

	mongoClient := &MongoDBClient{
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg,
	}

	basePort, err := strconv.Atoi(cfg.DB.MongoDB.Port)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("invalid MongoDB port: %w", err)
	}
	seedHosts := fmt.Sprintf("%s:%d,%s:%d,%s:%d",
		cfg.DB.MongoDB.Host, basePort,
		cfg.DB.MongoDB.Host, basePort+1,
		cfg.DB.MongoDB.Host, basePort+2,
	)
	uri := fmt.Sprintf("mongodb://%s", seedHosts)

	// Create a new client and connect to the server
	// WARN: BSON field 'insert.apiVersion' is an unknown field. (MongoDB (< 5.0))
	// serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	credential := options.Credential{Username: cfg.DB.MongoDB.User, Password: cfg.DB.MongoDB.Pass}

	opts := options.Client().
		ApplyURI(uri).
		// SetServerAPIOptions(serverAPI).
		SetReadPreference(readpref.Primary()).
		SetReplicaSet(cfg.DB.MongoDB.ReplicaSet).
		SetAuth(credential)

	client, err := mongo.Connect(opts)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Create a new database instance (dbname: offchain)
	db := client.Database(cfg.DB.MongoDB.DBName)
	mongoClient.db = db

	return mongoClient, nil
}

func (m *MongoDBClient) Start() {
	log.Info("Start MongoDB client")
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-m.ctx.Done():
				log.Info("MongoDB health check loop exiting due to context cancellation")
				return
			case <-ticker.C:
				_ = m.HealthCheck()
			}
		}
	}()
}

func (m *MongoDBClient) Stop() error {
	log.Info("Stop MongoDB client")
	return m.Close()
}

func (m *MongoDBClient) Close() error {
	m.cancel()

	client := m.db.Client()
	if err := client.Disconnect(m.ctx); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %w", err)
	}

	return nil
}

func (m *MongoDBClient) HealthCheck() error {
	if m == nil || m.db == nil {
		return fmt.Errorf("mongo client not initialized")
	}

	ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
	defer cancel()

	res := m.db.RunCommand(ctx, bson.D{{Key: "ping", Value: 1}})
	if err := res.Err(); err != nil {
		log.WithError(err).Warn("MongoDB health check failed")
		return err
	}

	log.Debug("MongoDB health check passed")
	return nil
}

func (m *MongoDBClient) HealthCheckWith(ctx context.Context) error {
	if m == nil || m.db == nil {
		return fmt.Errorf("mongo client not initialized")
	}

	res := m.db.RunCommand(ctx, bson.D{{Key: "ping", Value: 1}})
	if err := res.Err(); err != nil {
		log.WithError(err).Warn("MongoDB health check failed")
		return err
	}
	log.Debug("MongoDB health check passed")
	return nil
}

func (m *MongoDBClient) InsertOne(ctx context.Context, doc interface{}) (*mongo.InsertOneResult, error) {
	result, err := m.db.Collection(m.cfg.DB.MongoDB.Collection).InsertOne(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("failed to insert one document: %w", err)
	}

	// Inserted ID is result.InsertedID
	log.WithField("inserted_id", result.InsertedID).Debug("Inserted one document")
	return result, nil
}

func (m *MongoDBClient) FindOne(ctx context.Context, filter interface{}) (*mongo.SingleResult, error) {
	result := m.db.Collection(m.cfg.DB.MongoDB.Collection).FindOne(ctx, filter)
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("failed to find one document: %w", err)
	}

	log.WithField("result", result).Debug("Found one document")
	return result, nil
}

func (m *MongoDBClient) UploadToGridFS(ctx context.Context, filename string, data []byte, metadata bson.M) error {
	if m == nil || m.db == nil {
		return fmt.Errorf("mongo client not initialized")
	}

	// Create gridfs bucket
	bucket := m.db.GridFSBucket(
		options.GridFSBucket().SetName(m.cfg.DB.MongoDB.Collection),
		options.GridFSBucket().SetChunkSizeBytes(1024*255), // 255KB
	)

	var err error
	// Attach metadata if provided and upload with context
	if metadata != nil {
		uploadOpts := options.GridFSUpload().SetMetadata(metadata)
		_, err = bucket.UploadFromStream(ctx, filename, bytes.NewReader(data), uploadOpts)
	} else {
		_, err = bucket.UploadFromStream(ctx, filename, bytes.NewReader(data))
	}
	if err != nil {
		return fmt.Errorf("failed to upload to gridfs: %w", err)
	}

	log.WithField("filename", filename).Debug("Uploaded blob to GridFS")
	return nil
}
