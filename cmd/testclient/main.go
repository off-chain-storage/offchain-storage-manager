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

type config struct {
	addr    string
	path    string
	outPath string
	verify  bool
}

func main() {
	cfg := config{
		addr: "localhost:8080", // WARN: Please change Server Address
		// WARN: Alternative Path
		path:    "./data/data.log",
		outPath: "./data/output.json",
		verify:  true,
	}

	// gRPC Opts...
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxSize),
			grpc.MaxCallSendMsgSize(maxSize),
		),
	}

	// gRPC New Connection
	conn, err := grpc.NewClient(cfg.addr, dialOpts...)
	if err != nil {
		log.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// parse log by protojson (INPUT: txt file -> OUTPUT: protojson converted file)
	req, err := parseLog(cfg.path, cfg.outPath, cfg.verify)
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
