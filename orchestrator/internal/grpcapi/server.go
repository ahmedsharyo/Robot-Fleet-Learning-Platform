package grpcapi

// This package exposes the local HTTP API used by the runnable demo. The wire
// contract in proto/fleet/v1/fleet.proto is the canonical gRPC contract; clients
// can be generated with `make proto` once protoc and the Go/Python plugins exist.
import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/example/robot-fleet-learning-platform/orchestrator/internal/jobs"
	"github.com/example/robot-fleet-learning-platform/orchestrator/internal/metrics"
	"github.com/example/robot-fleet-learning-platform/orchestrator/internal/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	Store *storage.Store
	Lease time.Duration
	Mux   *http.ServeMux
}

func New(s *storage.Store) *Server {
	x := &Server{Store: s, Lease: 30 * time.Second, Mux: http.NewServeMux()}
	x.routes()
	return x
}
func (s *Server) routes() {
	s.Mux.Handle("/metrics", promhttp.Handler())
	s.Mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	s.Mux.HandleFunc("/v1/jobs", s.jobs)
	s.Mux.HandleFunc("/v1/jobs/", s.job)
	s.Mux.HandleFunc("/v1/workers/register", s.register)
	s.Mux.HandleFunc("/v1/workers/heartbeat", s.heartbeat)
	s.Mux.HandleFunc("/v1/workers/request", s.request)
	s.Mux.HandleFunc("/v1/jobs/start", s.start)
	s.Mux.HandleFunc("/v1/jobs/complete", s.complete)
	s.Mux.HandleFunc("/v1/jobs/fail", s.fail)
}
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	return json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v)
}
func (s *Server) jobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var in struct {
		IdempotencyKey string          `json:"idempotency_key"`
		Type           string          `json:"type"`
		Config         json.RawMessage `json:"config"`
		Priority       int             `json:"priority"`
		MaxRetries     int             `json:"max_retries"`
	}
	if decode(w, r, &in) != nil || in.IdempotencyKey == "" || in.Type == "" || len(in.Config) == 0 {
		http.Error(w, "invalid request", 400)
		return
	}
	var config any
	if json.Unmarshal(in.Config, &config) != nil {
		http.Error(w, "config must be JSON", 400)
		return
	}
	var idb [16]byte
	if _, e := rand.Read(idb[:]); e != nil {
		http.Error(w, "id generation failed", 500)
		return
	}
	now := time.Now().UTC()
	j := jobs.Job{ID: hex.EncodeToString(idb[:]), IdempotencyKey: in.IdempotencyKey, Type: in.Type, Config: in.Config, PayloadHash: jobs.HashPayload(in.Type, in.Config), Priority: in.Priority, Status: jobs.Pending, MaxRetries: in.MaxRetries, CreatedAt: now, UpdatedAt: now}
	got, e := s.Store.Submit(r.Context(), j)
	if errors.Is(e, jobs.ErrConflict) {
		http.Error(w, e.Error(), 409)
		return
	}
	if e == nil && got.ID == j.ID {
		_ = s.Store.Enqueue(r.Context(), j.ID)
		metrics.JobsSubmitted.WithLabelValues(j.Type).Inc()
	}
	if e != nil {
		http.Error(w, e.Error(), 500)
		return
	}
	writeJSON(w, 200, got)
}
func (s *Server) job(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/jobs/"), "/")
	id := parts[0]
	if len(parts) == 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
		if e := s.Store.Cancel(r.Context(), id); e != nil {
			http.Error(w, e.Error(), 409)
			return
		}
		writeJSON(w, 200, map[string]string{"status": "CANCELLED"})
		return
	}
	j, e := s.Store.Get(r.Context(), id)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, 200, j)
}
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		WorkerID     string   `json:"worker_id"`
		Capabilities []string `json:"capabilities"`
		CPUMillis    int32    `json:"cpu_millis"`
		MemoryBytes  int64    `json:"memory_bytes"`
	}
	if r.Method != http.MethodPost || decode(w, r, &in) != nil || in.WorkerID == "" {
		http.Error(w, "invalid request", 400)
		return
	}
	if e := s.Store.RegisterWorker(r.Context(), in.WorkerID, in.Capabilities, in.CPUMillis, in.MemoryBytes); e != nil {
		http.Error(w, e.Error(), 500)
		return
	}
	writeJSON(w, 200, map[string]any{"worker_id": in.WorkerID, "heartbeat_interval_seconds": 5})
}
func (s *Server) heartbeat(w http.ResponseWriter, r *http.Request) {
	var in struct {
		WorkerID string `json:"worker_id"`
	}
	if decode(w, r, &in) != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	if e := s.Store.Heartbeat(r.Context(), in.WorkerID); e != nil {
		http.Error(w, e.Error(), 404)
		return
	}
	writeJSON(w, 200, map[string]bool{"accepted": true})
}
func (s *Server) request(w http.ResponseWriter, r *http.Request) {
	var in struct {
		WorkerID     string   `json:"worker_id"`
		Capabilities []string `json:"capabilities"`
	}
	if decode(w, r, &in) != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	j, e := s.Store.LeaseNext(r.Context(), in.WorkerID, in.Capabilities, s.Lease)
	if e != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, 200, j)
}
func (s *Server) start(w http.ResponseWriter, r *http.Request) {
	var in struct {
		JobID      string `json:"job_id"`
		WorkerID   string `json:"worker_id"`
		LeaseToken string `json:"lease_token"`
	}
	if decode(w, r, &in) != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	if e := s.Store.SetRunning(r.Context(), in.JobID, in.WorkerID, in.LeaseToken); e != nil {
		http.Error(w, e.Error(), 409)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "RUNNING"})
}
func (s *Server) complete(w http.ResponseWriter, r *http.Request) {
	var in struct {
		JobID      string          `json:"job_id"`
		WorkerID   string          `json:"worker_id"`
		LeaseToken string          `json:"lease_token"`
		Artifacts  []jobs.Artifact `json:"artifacts"`
		Result     json.RawMessage `json:"result"`
	}
	if decode(w, r, &in) != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	if e := s.Store.Finish(r.Context(), in.JobID, in.WorkerID, in.LeaseToken, in.Artifacts, string(in.Result)); e != nil {
		http.Error(w, e.Error(), 409)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "SUCCEEDED"})
}
func (s *Server) fail(w http.ResponseWriter, r *http.Request) {
	var in struct {
		JobID      string `json:"job_id"`
		WorkerID   string `json:"worker_id"`
		LeaseToken string `json:"lease_token"`
		Error      string `json:"error"`
	}
	if decode(w, r, &in) != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	if e := s.Store.Fail(r.Context(), in.JobID, in.WorkerID, in.LeaseToken, in.Error); e != nil {
		http.Error(w, e.Error(), 409)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "RETRY_WAIT"})
}
func (s *Server) Recover(ctx context.Context) error { return s.Store.Recover(ctx) }
func (s *Server) Maintenance(ctx context.Context) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if e := s.Store.RecoverExpired(ctx); e != nil {
				continue
			}
		}
	}
}
