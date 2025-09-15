package db

import (
	"context"
	"time"

	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/types"
	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/util"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var log = util.NewLogger("db")

type DBService struct {
	ctx    context.Context
	cancel context.CancelFunc
	Mongo  *MongoDBClient
}

func NewDBService(cfg *types.Config) (*DBService, error) {
	// Context for upper service
	ctx, cancel := context.WithCancel(context.Background())

	// Context for init timeout (15 seconds) - health check
	initCtx, initCancel := context.WithTimeout(ctx, 15*time.Second)
	defer initCancel()

	// MongoDB Service
	mongoClient, err := NewMongoDBClient(ctx, cfg)
	if err != nil {
		cancel()
		return nil, err
	}

	// Health Check
	if err := mongoClient.HealthCheckWith(initCtx); err != nil {
		_ = mongoClient.Close()
		cancel()
		return nil, err
	}

	// DB Service
	d := &DBService{
		ctx:    ctx,
		cancel: cancel,
		Mongo:  mongoClient,
	}

	return d, nil
}

func (d *DBService) Start() {
	log.Info("Starting DB service")
	d.Mongo.Start()
}

func (d *DBService) Stop() error {
	log.Info("Stopping DB service")
	return d.Close()
}

func (d *DBService) Close() error {
	log.Info("Closing DB service")
	d.cancel()
	d.Mongo.Close()

	return nil
}

// Implement iface.MongoDB by delegating to the Mongo Service
func (d *DBService) InsertOne(ctx context.Context, doc interface{}) (*mongo.InsertOneResult, error) {
	return d.Mongo.InsertOne(ctx, doc)
}

func (d *DBService) FindOne(ctx context.Context, filter interface{}) (*mongo.SingleResult, error) {
	return d.Mongo.FindOne(ctx, filter)
}

func (d *DBService) UploadToGridFS(ctx context.Context, filename string, data []byte, metadata bson.M) error {
	return d.Mongo.UploadToGridFS(ctx, filename, data, metadata)
}
