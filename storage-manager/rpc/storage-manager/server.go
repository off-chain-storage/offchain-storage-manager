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

	// Compute stats for logging: full block size, extracted info size, and savings
	blockSize := len(raw)
	extractedSize, savingsPct := getExtractionStats(in, blockSize)
	log.WithFields(map[string]interface{}{
		"block_size":       util.HumanBytes(int64(blockSize)),
		"extracted_size":   util.HumanBytes(int64(extractedSize)),
		"data_savings_pct": fmt.Sprintf("%.3f%%", savingsPct),
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

func getExtractionStats(in *storagemgrPB.ExecutableConsensusOutput, blockSize int) (extractedSize int, savingsPct float64) {
	// Calculate total extracted info size (sum of tx bytes) and savings ratio
	var totalTxBytes int
	for _, batch := range in.GetData() {
		for _, tx := range batch.GetData() {
			if bs, e := (proto.MarshalOptions{Deterministic: true}).Marshal(tx); e == nil {
				totalTxBytes += len(bs)
			}
		}
	}

	if blockSize > 0 {
		ratio := 1.0 - (float64(totalTxBytes) / float64(blockSize))
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		savingsPct = ratio * 100.0
	}
	return totalTxBytes, savingsPct
}
