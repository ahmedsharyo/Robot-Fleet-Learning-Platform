# Technical Report

## Scope and evidence

This report describes the initial code in this repository. No end-to-end run, MuJoCo collection, training run, Docker deployment, or performance experiment has been completed in the current environment. Python `pytest` is unavailable and Go dependency retrieval is blocked by offline networking; avoid interpreting this report as measured validation.

## Design

Go owns job state in SQLite WAL. Attempt-specific lease tokens reject stale completion writes. Workers initiate polling. Episode artifacts are stored separately in content-hashed Parquet partitions with a manifest. BC uses an episode-level split; PPO collects fresh environment data in a single process. Evaluation helper supports three fixed, distinct seeds and reports return and timing without inventing a task success metric.

## Results

No measured simulation scaling, scheduler latency, recovery time, or learning comparison is available. The scheduling harness can record individual request latency when run against a live orchestrator. Statistical aggregation and the 1/2/4/8 worker experiment remain to be completed.

## Reliability and bottlenecks

Lease fencing and SQLite persistence are implemented in code; crash recovery tests are limited. The main expected bottlenecks are CPU contention, Python startup, SQLite's single writer, serialization, filesystem throughput, and the shared-volume model. These are hypotheses until measured.

## Limitations and next work

Finish the gRPC server/generated clients, lease-renewing worker, typed job handlers, resource admission and bounded queues, integration/failure-injection tests, worker metrics, dashboards, and full benchmark suite. The platform is local-only and has no physical robot validation.
