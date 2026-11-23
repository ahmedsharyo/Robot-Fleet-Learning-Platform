SHELL := /bin/sh
PYTHON ?= python3
GO ?= go
.PHONY: setup setup-python setup-go proto test test-go test-python lint up down demo benchmark report
setup: setup-go setup-python
setup-go:
	cd orchestrator && $(GO) mod download
setup-python:
	$(PYTHON) -m venv .venv
	.venv/bin/pip install -r requirements.txt
proto:
	@command -v protoc >/dev/null || (echo 'protoc is required; install with brew install protobuf' >&2; exit 1)
	@echo 'Install protoc-gen-go, protoc-gen-go-grpc, and grpcio-tools; see docs/protocol.md'
	protoc -I proto --go_out=orchestrator/gen --go_opt=module=github.com/example/robot-fleet-learning-platform/orchestrator --go-grpc_out=orchestrator/gen --go-grpc_opt=module=github.com/example/robot-fleet-learning-platform/orchestrator proto/fleet/v1/fleet.proto
	$(PYTHON) -m grpc_tools.protoc -I proto --python_out=workers/generated --grpc_python_out=workers/generated proto/fleet/v1/fleet.proto
test: test-go test-python
test-go:
	cd orchestrator && $(GO) test -race ./...
test-python:
	$(PYTHON) -m pytest -q
lint:
	cd orchestrator && $(GO) vet ./...
	$(PYTHON) -m compileall -q datasets workers learning benchmarks
up:
	docker compose up --build
down:
	docker compose down
demo:
	$(PYTHON) scripts/demo.py
benchmark:
	$(PYTHON) benchmarks/scheduling/run.py --jobs 30
report:
	$(PYTHON) benchmarks/analysis/report.py reports/benchmark.json
