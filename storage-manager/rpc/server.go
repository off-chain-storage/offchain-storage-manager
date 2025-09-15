package rpc

import (
	"context"
	"net"

	storagemgrPB "github.com/off-chain-storage/offchain-storage-manager/proto"
	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/db"
	storagemgr "github.com/off-chain-storage/offchain-storage-manager/storage-manager/rpc/storage-manager"
	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/types"
	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var log = util.NewLogger("grpc.server")

type ServerService struct {
	ctx        context.Context
	cancel     context.CancelFunc
	cfg        *types.Config
	mongo      db.AccessMongoDB
	listener   net.Listener
	grpcServer *grpc.Server
	grpcClient map[net.Addr]bool
}

func NewGRPCServer(ctx context.Context, cfg *types.Config, mongo db.AccessMongoDB) *ServerService {
	ctx, cancel := context.WithCancel(ctx)

	g := &ServerService{
		ctx:        ctx,
		cancel:     cancel,
		cfg:        cfg,
		mongo:      mongo,
		grpcClient: make(map[net.Addr]bool),
	}

	address := cfg.GRPC.ListenAddr
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.WithError(err).Fatalln("Could not listen to port in Start()", address)
	}

	g.listener = lis
	log.WithField("Address", address).Info("gRPC server listening on port")

	maxMsgSize := cfg.GRPC.MaxMsgSize
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(maxMsgSize),
	}

	g.grpcServer = grpc.NewServer(opts...)

	return g
}

func (g *ServerService) Start() {
	propagationManagerServer := &storagemgr.Manager{
		Ctx:   g.ctx,
		Mongo: g.mongo,
	}
	storagemgrPB.RegisterStorageManagerServer(g.grpcServer, propagationManagerServer)
	reflection.Register(g.grpcServer)

	go func() {
		if g.listener != nil {
			if err := g.grpcServer.Serve(g.listener); err != nil {
				log.WithError(err).Error("gRPC server failed to serve")
			}
		}
	}()
}

func (rs *ServerService) Stop() error {
	rs.cancel()
	if rs.listener != nil {
		rs.grpcServer.GracefulStop()
		log.Debugln("gRPC server stopped")
	}
	return nil
}
