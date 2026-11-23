# Protocol

`proto/fleet/v1/fleet.proto` is the canonical contract for registration, heartbeat, polling, start, completion, failure, lease renewal, cancellation, and status reads. Large trajectory arrays do not belong in RPC messages; artifacts are shared-storage URIs with SHA-256 references.

Generate bindings after installing tools:

```bash
brew install protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.35.1
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
python -m pip install grpcio grpcio-tools
export PATH="$(go env GOPATH)/bin:$PATH"
make proto
```

The current runnable control plane uses HTTP JSON and does not yet implement gRPC deadlines or the protobuf service. This is a known gap; do not describe the current endpoint as gRPC.
