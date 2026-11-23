# Architecture

The Go process owns job metadata and worker liveness. SQLite runs in WAL mode and uses a single connection for predictable local transactions. Jobs move from pending to queued, leased, running, and a terminal state; recoverable errors enter retry-wait. Each lease carries a fresh token and commits require the active worker/token pair and an unexpired lease. Workers poll so they need no inbound endpoint. The data plane uses shared filesystem paths; only artifact references and metadata belong in job results.

The implementation is currently a JSON/HTTP API used to make the local demo runnable. `proto/fleet/v1/fleet.proto` defines the planned gRPC API, but generated bindings/server registration are not yet implemented. SQLite is behind the `storage.Store` boundary, but a PostgreSQL backend is not implemented.

## Planes

- Control: Go API, state machine, SQLite, polling leases.
- Data: shared filesystem, validated Parquet dataset versions, model checkpoints.
- Observability: `/metrics`, Prometheus scrape config, Grafana datasource provisioning.

## Recovery and consistency

Lease expiry fences late success. Lease recovery either queues another attempt or marks the job failed after retries. Artifact publication stages data in a sibling directory and renames it into place, preventing readers from seeing a partial manifest. Dataset paths must be unique and immutable. A worker can still leave orphan staging/artifact files after a crash; garbage collection is future work.
