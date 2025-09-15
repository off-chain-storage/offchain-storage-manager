FROM golang:1.24.5-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN  go build \
    -ldflags="-s -w \
    -X github.com/off-chain-storage/offchain-storage-manager/storage-manager/types.Version=dev \
    -X github.com/off-chain-storage/offchain-storage-manager/storage-manager/types.GitCommitHash=dev" \
    -o storage-manager ./cmd/storage-manager

FROM alpine:latest

WORKDIR /app

RUN mkdir -p /etc/storage-manager

RUN apk --no-cache add ca-certificates

COPY --from=builder /build/storage-manager /app/
COPY --from=builder /build/config.yaml /etc/storage-manager/config.yaml

ENV STORAGE_MANAGER_CONFIG=/etc/storage-manager/config.yaml

ENTRYPOINT ["/app/storage-manager"]
CMD ["start"]
