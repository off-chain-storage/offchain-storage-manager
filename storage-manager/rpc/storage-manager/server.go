package rpc_storagemgr

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	storagemgrPB "github.com/off-chain-storage/offchain-storage-manager/proto"
	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/db"
	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/util"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/proto"
)

var log = util.NewLogger("grpc.server.storage-manager")

type Manager struct {
	Ctx   context.Context
	Mongo db.AccessMongoDB
}

// gRPC Server Handler for StorageManager.StoreBlocks
func (m *Manager) StoreBlocks(ctx context.Context, in *storagemgrPB.ExecutableConsensusOutput) (*storagemgrPB.StoreResponse, error) {
	log.WithField("batches", len(in.GetData())).Info("StoreBlocks request received")

	// Deterministically serialize the whole message
	raw, err := (proto.MarshalOptions{Deterministic: true}).Marshal(in)
	if err != nil {
		return &storagemgrPB.StoreResponse{
			Success: false,
			Message: fmt.Sprintf("marshal failed: %v", err),
		}, nil
	}

	totalSize, avgTxPerBatch, avgTxSize, pct := getStats(in)
	log.WithFields(map[string]interface{}{
		"in_size":          util.HumanBytes(int64(totalSize)),
		"avg_tx_per_batch": fmt.Sprintf("%.3f", avgTxPerBatch),
		"avg_tx_size":      util.HumanBytes(int64(avgTxSize)),
		"avg_tx_pct":       fmt.Sprintf("%.3f%%", pct),
	}).Info("StoreBlocks stats")

	// Generate a stable CID-like identifier by SHA256
	sum := sha256.Sum256(raw)
	b32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:])

	cid := "cidv1-raw-sha256-" + strings.ToLower(b32)

	// Store depending on size (MongoDB document limit ~=16MB)
	const mongoMaxDoc = 16 * 1024 * 1024 // 16MB
	size := len(raw)

	if size <= mongoMaxDoc {
		// Store as a single document in the configured collection
		doc := bson.M{
			"_id":        cid,
			"size":       util.HumanBytes(int64(size)),
			"created_at": time.Now().UTC(),
			"raw":        raw,
		}

		// MONGODB: INSERT to Collection
		if _, err := m.Mongo.InsertOne(ctx, doc); err != nil {
			// if key(block cid) already exists, treat as success and return CID
			if db.IsDuplicateKey(err) {
				log.WithField("cid", cid).Info("Block already exists, returning existing CID")
			} else {
				return &storagemgrPB.StoreResponse{
					Success: false,
					Message: fmt.Sprintf("insert failed: %v", err),
				}, nil
			}
		}
		return &storagemgrPB.StoreResponse{
			Success: true,
			Message: cid,
		}, nil
	}

	// if larger than 16MB: use GridFS
	if concrete, ok := m.Mongo.(*db.MongoDBClient); ok {
		// MONGODB: UPLOAD to GridFS
		if err := concrete.UploadToGridFS(ctx, cid, raw, bson.M{"size": size, "created_at": time.Now().UTC()}); err != nil {
			return &storagemgrPB.StoreResponse{
				Success: false,
				Message: fmt.Sprintf("gridfs upload failed: %v", err),
			}, nil
		}
		return &storagemgrPB.StoreResponse{
			Success: true,
			Message: cid,
		}, nil
	}

	return &storagemgrPB.StoreResponse{
		Success: false,
		Message: "gridfs not available for large payloads",
	}, nil
}

func getStats(in *storagemgrPB.ExecutableConsensusOutput) (totalSize int, avgTxPerBatch float64, avgTxSize float64, pct float64) {
	// Stats: total message size and TransactionSigned distribution
	var totalTxCount int
	var totalTxBytes int

	for _, batch := range in.GetData() {
		for _, tx := range batch.GetData() {
			totalTxCount++
			if bs, e := (proto.MarshalOptions{Deterministic: true}).Marshal(tx); e == nil {
				totalTxBytes += len(bs)
			}
		}
	}
	batchCount := len(in.GetData())

	if batchCount > 0 {
		avgTxPerBatch = float64(totalTxCount) / float64(batchCount)
	}

	if totalTxCount > 0 {
		avgTxSize = float64(totalTxBytes) / float64(totalTxCount)
	}

	if totalSize > 0 {
		pct = (avgTxSize / float64(totalSize)) * 100.0
	}
	return totalSize, avgTxPerBatch, avgTxSize, pct
}
