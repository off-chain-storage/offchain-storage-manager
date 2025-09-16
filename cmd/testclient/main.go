package main

import (
	"context"
	"time"

	storagemgrPB "github.com/off-chain-storage/offchain-storage-manager/proto"

	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/util"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var log = util.NewLogger("grpc.client")

// gRPC msg size limit - 1GB
const maxSize = 1024 * 1024 * 1024

type Config struct {
	Addr    string
	Path    string
	OutPath string
	Verify  bool
}

func main() {
	cfg := Config{
		Addr: "localhost:8080", // WARN: Please change Server Address
		// WARN: Alternative Path
		Path:    "./data/data.log",
		OutPath: "./data/output.json",
		Verify:  true,
	}

	// gRPC Opts...
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxSize),
			grpc.MaxCallSendMsgSize(maxSize),
		),
	}

	// gRPC New Connection
	conn, err := grpc.NewClient(cfg.Addr, dialOpts...)
	if err != nil {
		log.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// parse log by protojson (INPUT: txt file -> OUTPUT: protojson converted file)
	req, err := parseLog(cfg.Path, cfg.OutPath, cfg.Verify)
	if err != nil {
		log.Fatalf("parse: %v", err)
	}

	// gRPC NewClient
	c := storagemgrPB.NewStorageManagerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// RPC - StoreBlocks
	resp, err := c.StoreBlocks(ctx, req)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("success=%v msg=%s", resp.GetSuccess(), resp.GetMessage())
}
