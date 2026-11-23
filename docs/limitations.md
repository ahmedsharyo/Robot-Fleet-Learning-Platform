# Limitations

- The control API is HTTP JSON; protobuf bindings and a gRPC listener are not implemented yet.
- Resource matching fields are recorded but CPU/memory admission control and bounded queue rejection are not implemented.
- Worker heartbeats are sent between jobs; long-running collection does not yet renew its lease in a background task, so jobs exceeding the lease can become stale.
- Collection produces a random baseline only. It is not an expert demonstration.
- Worker dispatch currently performs collection only. Other job types do not yet invoke BC, PPO, validation, or evaluation handlers.
- The PPO trainer is single-process and its rollout/update path is a compact baseline, not a performance-validated implementation.
- Grafana provisioning has a data source but no finished dashboards; Python worker metrics are not exposed.
- Scalability, recovery, backpressure and learning benchmarks have not been run. No results are claimed.
- SQLite and shared local filesystem are single-host choices. There is no multi-host artifact transport, authn/z, TLS, or PostgreSQL backend.
- No physical robot, sim-to-real transfer, production deployment, or fleet-scale operation has been tested.
