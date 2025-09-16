# Storage-Manager Test Client (gRPC)

ExecutableConsensusOutput 더미 데이터를 오프체인 스토리지 매니저(gRPC 서버) 인스턴스로 전송하는 테스트용 gRPC 클라이언트

## 테스트용 설정 값

`cmd/testclient/main.go`의 `config` struct는 다음과 같이 정의됨

```go
type config struct {
    addr    string
    path    string
    outPath string
    verify  bool
}
```
- `addr`: gRPC 서버 엔드포인트 주소. Default: `localhost:8080`
- `path`: 입력 더미 로그 파일 경로. 샘플 파일은 `cmd/testclient/data/dummy_data_1.log` 등을 사용
- `outPath`: 변환된 protojson 파일이 저장될 경로
- `verify`: `true`일 때 JSON → protobuf 역직렬화 검증을 수행
- 새로운 로그 테스트 시 `cmd/testclient/data` 폴더에 파일을 추가 후 `Path`를 해당 경로로 지정

## 실행 방법

```bash
go run ./cmd/testclient # Run in project root
```

## 출력 로그
```bash
INFO[0000] wrote ./data/output.json (5752722 bytes)      prefix=grpc.client
INFO[0000] verify OK: batches=184, txs_in_first=1        prefix=grpc.client
INFO[0000] success=true msg=cidv1-raw-sha256-murnqpd5hlqfyj6c6532nbqqahgds3kh6jtswwd2cnoproop3y3a  prefix=grpc.client
```