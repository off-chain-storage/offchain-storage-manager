package storagemgr

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/db"
	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/rpc"
	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/runtime"
	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/runtime/errors"
	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/types"
	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/util"
)

var log = util.NewLogger("storagemanager")

type StorageManager struct {
	cfg      *types.Config
	ctx      context.Context
	cancel   context.CancelFunc
	services *runtime.ServiceRegistry
	lock     sync.RWMutex
	stop     chan struct{}
	db       db.DB
}

func NewManager() (*StorageManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	registry := runtime.NewServiceRegistry()
	cfg, err := types.LoadManagerConfig()
	if err != nil {
		log.WithError(err).Fatal("failed to load configuration while creating storage manager instance")
	}

	s := &StorageManager{
		ctx:      ctx,
		cancel:   cancel,
		cfg:      cfg,
		services: registry,
		stop:     make(chan struct{}),
	}

	// Registry Base Services
	if err := s.registerBaseServices(ctx); err != nil {
		return nil, errors.WrapWithMessage(err, "could not start base modules")
	}

	return s, nil
}

func (s *StorageManager) registerBaseServices(ctx context.Context) error {
	log.Debugln("Registering base services")

	// Register DB Service
	log.Debugln("[Base] Registering DB Service")
	dbService, err := db.NewDBService(s.cfg)
	if err != nil {
		log.WithError(err).Fatal("failed to initialize db service")
	}
	if err := s.services.RegisterService(dbService); err != nil {
		log.WithError(err).Warn("failed to register db service")
		return err
	}
	s.db = dbService

	// Register GRPC Service - Depend on DB Service (only access mongoDB)
	log.Debugln("[Base] Registering GRPC Service")
	grpcService := rpc.NewGRPCServer(ctx, s.cfg, dbService.Mongo)
	if err := s.services.RegisterService(grpcService); err != nil {
		log.WithError(err).Warn("failed to register grpc service")
		return err
	}

	return nil
}

func (s *StorageManager) Start() error {
	s.lock.Lock()

	log.Info("Starting storage manager...")
	s.services.StartAll()
	stop := s.stop

	s.lock.Unlock()

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc

		log.Info("Received shutdown signal, initiating graceful shutdown")

		go s.Close()

		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				log.WithField("remaining_attempts", i-1).Warn("Shutdown in progress, additional interrupts will force immediate termination")
			}
		}
		log.Fatal("Force termination triggered by multiple interrupts")
	}()

	<-stop
	return nil

}

func (s *StorageManager) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	log.Info("Closing storage manager...")
	s.services.StopAll()
	s.cancel()

	close(s.stop)
	log.Info("Storage manager closed")
}
