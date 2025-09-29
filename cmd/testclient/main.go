package main

import (
	"context"
	"strconv"
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
	path    []string
	outPath []string
	verify  bool
}

func main() {
	cfg := config{
		addr: "localhost:8080", // WARN: Please change Server Address
		// WARN: Alternative Path
		// path:    "./cmd/testclient/data/dummy_data_1.log",
		path: []string{
			"./cmd/testclient/data/dummy_data_1.log",
			"./cmd/testclient/data/dummy_data_2.log",
			"./cmd/testclient/data/dummy_data_3.log",
			"./cmd/testclient/data/dummy_data_4.log",
			"./cmd/testclient/data/dummy_data_5.log",
			"./cmd/testclient/data/dummy_data_6.log",
			"./cmd/testclient/data/dummy_data_7.log",
			"./cmd/testclient/data/dummy_data_8.log",
			"./cmd/testclient/data/dummy_data_9.log",
			"./cmd/testclient/data/dummy_data_10.log",
		},
		outPath: []string{
			"./cmd/testclient/data/output_1.json",
			"./cmd/testclient/data/output_2.json",
			"./cmd/testclient/data/output_3.json",
			"./cmd/testclient/data/output_4.json",
			"./cmd/testclient/data/output_5.json",
			"./cmd/testclient/data/output_6.json",
			"./cmd/testclient/data/output_7.json",
			"./cmd/testclient/data/output_8.json",
			"./cmd/testclient/data/output_9.json",
			"./cmd/testclient/data/output_10.json",
		},
		verify: true,
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

	// gRPC NewClient
	c := storagemgrPB.NewStorageManagerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	total_saving_pct := 0.0

	// RPC - StoreBlocks
	for i := 0; i < 10; i++ {
		path := cfg.path[i]
		outPath := cfg.outPath[i]

		// parse log by protojson (INPUT: txt file -> OUTPUT: protojson converted file)
		req, err := parseLog(path, outPath, cfg.verify)
		if err != nil {
			log.Fatalf("parse: %v", err)
		}

		resp, err := c.StoreBlocks(ctx, req)
		if err != nil {
			log.Fatal(err)
		}

		temp, err := strconv.ParseFloat(resp.GetMessage(), 64)
		if err != nil {
			log.Fatal(err)
		}
		total_saving_pct += temp
	}

	log.Infof("average_data_saving_pct: %v", total_saving_pct/10.0)
}
