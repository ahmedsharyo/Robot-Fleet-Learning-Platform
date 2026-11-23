package storage

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/example/robot-fleet-learning-platform/orchestrator/internal/jobs"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, e := Open(filepath.Join(t.TempDir(), "jobs.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func newJob() jobs.Job {
	return jobs.Job{ID: "j1", IdempotencyKey: "k1", Type: "COLLECT_EPISODES", Config: json.RawMessage(`{"episodes":1}`), PayloadHash: jobs.HashPayload("COLLECT_EPISODES", json.RawMessage(`{"episodes":1}`)), Status: jobs.Pending, MaxRetries: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
}
func TestIdempotency(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	j := newJob()
	if _, e := s.Submit(ctx, j); e != nil {
		t.Fatal(e)
	}
	j.ID = "different"
	got, e := s.Submit(ctx, j)
	if e != nil || got.ID != "j1" {
		t.Fatalf("got %+v, %v", got, e)
	}
	j.PayloadHash = "different"
	if _, e = s.Submit(ctx, j); !errors.Is(e, jobs.ErrConflict) {
		t.Fatalf("expected conflict, got %v", e)
	}
}
func TestStaleLeaseRejected(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	j := newJob()
	if _, e := s.Submit(ctx, j); e != nil {
		t.Fatal(e)
	}
	if e := s.Enqueue(ctx, j.ID); e != nil {
		t.Fatal(e)
	}
	if e := s.RegisterWorker(ctx, "w", []string{j.Type}, 1000, 1<<30); e != nil {
		t.Fatal(e)
	}
	leased, e := s.LeaseNext(ctx, "w", []string{j.Type}, time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SetRunning(ctx, j.ID, "w", leased.LeaseToken); e != nil {
		t.Fatal(e)
	}
	if e = s.Finish(ctx, j.ID, "w", "old-token", nil, ""); !errors.Is(e, jobs.ErrStaleLease) {
		t.Fatalf("expected stale lease, got %v", e)
	}
}
