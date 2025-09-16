# Offchain Storage Manager

## Main Components
- `cmd/storage-manager`: CLI entrypoint
- `storage-manager/db`: Database service layer with MongoDB integration
- `storage-manager/rpc`: gRPC server and `StoreBlocks` implementation
- `proto/`: StorageManager gRPC interface and message schemas
- `cmd/testclient`: gRPC client for testing (see separate README)

## Requirements
- Set `config.yaml`

- MongoDB (configured separately)

## Get Started
1. Clone the repository:
```bash
git clone https://github.com/off-chain-storage/offchain-storage-manager.git
cd offchain-storage-manager
```

2. Set up environment variables
- Create a `config.yaml` file in root directory
```yaml
log:
  level: info
  format: text

grpc:
  listen_addr: 0.0.0.0:8080
  max_msg_size: 1048576 # bytes, Default 1MB
  timeout: 10s

db:
  mongodb:
    host: mongodb # change address as needed
    port: 27017
    replica_set: rs0
    dbname: offchain
    collection: blocks
    user: root
    password: root
```

3. Run w/ Container
```bash
docker compose up --build storage-manager
```

## Interface
### gRPC Service
- **Service**: `storagemgr.StorageManager`
- **Method(RPC)**: `StoreResponse`

### Request/Response Schema
**Request schema**: see `proto/common.proto` 

**Response**:
- `success`: boolean flag
- `message`: CID on success, error message on failure


## Test
- Before running, configure the `Config` values in `cmd/testclient/main.go`.  
- Use `cmd/testclient` to send sample logs located in `cmd/testclient/data/*.log`.
- For more details, see `cmd/testclient/README.md`