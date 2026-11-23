package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/example/robot-fleet-learning-platform/orchestrator/internal/jobs"
	_ "github.com/mattn/go-sqlite3"
)

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	schema := `CREATE TABLE IF NOT EXISTS jobs (id TEXT PRIMARY KEY, idem TEXT NOT NULL UNIQUE, payload_hash TEXT NOT NULL, type TEXT NOT NULL, config TEXT NOT NULL, priority INTEGER NOT NULL, status TEXT NOT NULL, attempt INTEGER NOT NULL, max_retries INTEGER NOT NULL, worker_id TEXT NOT NULL DEFAULT '', lease_token TEXT NOT NULL DEFAULT '', lease_until INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, error TEXT NOT NULL DEFAULT '', artifacts TEXT NOT NULL DEFAULT '[]'); CREATE INDEX IF NOT EXISTS jobs_queue ON jobs(status, priority DESC, created_at); CREATE TABLE IF NOT EXISTS workers (id TEXT PRIMARY KEY, capabilities TEXT NOT NULL, cpu_millis INTEGER NOT NULL, memory_bytes INTEGER NOT NULL, last_seen INTEGER NOT NULL);`
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db}, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Submit(ctx context.Context, j jobs.Job) (jobs.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return j, err
	}
	defer tx.Rollback()
	var old jobs.Job
	old, err = scanJob(tx.QueryRowContext(ctx, selectJob+` WHERE idem=?`, j.IdempotencyKey))
	if err == nil {
		if old.PayloadHash != j.PayloadHash {
			return j, jobs.ErrConflict
		}
		return old, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return j, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO jobs(id,idem,payload_hash,type,config,priority,status,attempt,max_retries,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, j.ID, j.IdempotencyKey, j.PayloadHash, j.Type, string(j.Config), j.Priority, j.Status, j.Attempt, j.MaxRetries, j.CreatedAt.UnixNano(), j.UpdatedAt.UnixNano())
	if err != nil {
		return j, err
	}
	return j, tx.Commit()
}

const selectJob = `SELECT id,idem,payload_hash,type,config,priority,status,attempt,max_retries,worker_id,lease_token,lease_until,created_at,updated_at,error,artifacts FROM jobs`

func scanJob(row interface{ Scan(...any) error }) (jobs.Job, error) {
	var j jobs.Job
	var cfg, artifacts string
	var lease, created, updated int64
	err := row.Scan(&j.ID, &j.IdempotencyKey, &j.PayloadHash, &j.Type, &cfg, &j.Priority, &j.Status, &j.Attempt, &j.MaxRetries, &j.WorkerID, &j.LeaseToken, &lease, &created, &updated, &j.Error, &artifacts)
	if err != nil {
		return j, err
	}
	j.Config = json.RawMessage(cfg)
	j.Artifacts = []jobs.Artifact{}
	_ = json.Unmarshal([]byte(artifacts), &j.Artifacts)
	if lease > 0 {
		j.LeaseUntil = time.Unix(0, lease).UTC()
	}
	j.CreatedAt = time.Unix(0, created).UTC()
	j.UpdatedAt = time.Unix(0, updated).UTC()
	return j, nil
}
func (s *Store) Get(ctx context.Context, id string) (jobs.Job, error) {
	return scanJob(s.db.QueryRowContext(ctx, selectJob+` WHERE id=?`, id))
}
func (s *Store) Enqueue(ctx context.Context, id string) error {
	_, e := s.db.ExecContext(ctx, `UPDATE jobs SET status='QUEUED',updated_at=? WHERE id=? AND status='PENDING'`, time.Now().UnixNano(), id)
	return e
}
func (s *Store) RegisterWorker(ctx context.Context, id string, caps []string, cpu int32, mem int64) error {
	b, _ := json.Marshal(caps)
	_, e := s.db.ExecContext(ctx, `INSERT INTO workers(id,capabilities,cpu_millis,memory_bytes,last_seen) VALUES(?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET capabilities=excluded.capabilities,cpu_millis=excluded.cpu_millis,memory_bytes=excluded.memory_bytes,last_seen=excluded.last_seen`, id, string(b), cpu, mem, time.Now().UnixNano())
	return e
}
func (s *Store) Heartbeat(ctx context.Context, id string) error {
	r, e := s.db.ExecContext(ctx, `UPDATE workers SET last_seen=? WHERE id=?`, time.Now().UnixNano(), id)
	if e != nil {
		return e
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return fmt.Errorf("unknown worker %s", id)
	}
	return nil
}
func (s *Store) LeaseNext(ctx context.Context, worker string, caps []string, lease time.Duration) (jobs.Job, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return jobs.Job{}, e
	}
	defer tx.Rollback()
	rows, e := tx.QueryContext(ctx, `SELECT id FROM jobs WHERE status='QUEUED' ORDER BY priority DESC,created_at LIMIT 100`)
	if e != nil {
		return jobs.Job{}, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return jobs.Job{}, e
		}
		ids = append(ids, id)
	}
	rows.Close()
	capSet := map[string]bool{}
	for _, c := range caps {
		capSet[c] = true
	}
	for _, id := range ids {
		j, er := scanJob(tx.QueryRowContext(ctx, selectJob+` WHERE id=?`, id))
		if er != nil {
			return jobs.Job{}, er
		}
		if !capSet[j.Type] {
			continue
		}
		token := fmt.Sprintf("%s-%d-%d", id, j.Attempt+1, time.Now().UnixNano())
		until := time.Now().Add(lease)
		res, er := tx.ExecContext(ctx, `UPDATE jobs SET status='LEASED',attempt=attempt+1,worker_id=?,lease_token=?,lease_until=?,updated_at=? WHERE id=? AND status='QUEUED'`, worker, token, until.UnixNano(), time.Now().UnixNano(), id)
		if er != nil {
			return jobs.Job{}, er
		}
		n, _ := res.RowsAffected()
		if n == 1 {
			if er = tx.Commit(); er != nil {
				return jobs.Job{}, er
			}
			return s.Get(ctx, id)
		}
	}
	return jobs.Job{}, sql.ErrNoRows
}
func (s *Store) SetRunning(ctx context.Context, id, worker, token string) error {
	r, e := s.db.ExecContext(ctx, `UPDATE jobs SET status='RUNNING',updated_at=? WHERE id=? AND worker_id=? AND lease_token=? AND status='LEASED' AND lease_until>?`, time.Now().UnixNano(), id, worker, token, time.Now().UnixNano())
	if e != nil {
		return e
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return jobs.ErrStaleLease
	}
	return nil
}
func (s *Store) Renew(ctx context.Context, id, worker, token string, d time.Duration) error {
	r, e := s.db.ExecContext(ctx, `UPDATE jobs SET lease_until=?,updated_at=? WHERE id=? AND worker_id=? AND lease_token=? AND status IN ('LEASED','RUNNING') AND lease_until>?`, time.Now().Add(d).UnixNano(), time.Now().UnixNano(), id, worker, token, time.Now().UnixNano())
	if e != nil {
		return e
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return jobs.ErrStaleLease
	}
	return nil
}
func (s *Store) Finish(ctx context.Context, id, worker, token string, arts []jobs.Artifact, result string) error {
	b, e := json.Marshal(arts)
	if e != nil {
		return e
	}
	res, e := s.db.ExecContext(ctx, `UPDATE jobs SET status='SUCCEEDED',artifacts=?,error='',updated_at=? WHERE id=? AND worker_id=? AND lease_token=? AND status='RUNNING' AND lease_until>?`, string(b), time.Now().UnixNano(), id, worker, token, time.Now().UnixNano())
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return jobs.ErrStaleLease
	}
	return nil
}
func (s *Store) Fail(ctx context.Context, id, worker, token, msg string) error {
	now := time.Now()
	res, e := s.db.ExecContext(ctx, `UPDATE jobs SET status=CASE WHEN attempt<=max_retries THEN 'RETRY_WAIT' ELSE 'FAILED' END,error=?,lease_until=?,updated_at=? WHERE id=? AND worker_id=? AND lease_token=? AND status IN ('LEASED','RUNNING') AND lease_until>?`, msg, now.Add(time.Second).UnixNano(), now.UnixNano(), id, worker, token, now.UnixNano())
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return jobs.ErrStaleLease
	}
	return nil
}
func (s *Store) Recover(ctx context.Context) error {
	now := time.Now().UnixNano()
	_, e := s.db.ExecContext(ctx, `UPDATE jobs SET status=CASE WHEN attempt<=max_retries THEN 'RETRY_WAIT' ELSE 'FAILED' END,error='lease expired',updated_at=? WHERE status IN ('LEASED','RUNNING') AND lease_until<?`, now, now)
	if e != nil {
		return e
	}
	_, e = s.db.ExecContext(ctx, `UPDATE jobs SET status='QUEUED',updated_at=? WHERE status='RETRY_WAIT' AND lease_until<?`, now, now)
	if e != nil {
		return e
	}
	_, e = s.db.ExecContext(ctx, `UPDATE jobs SET status='QUEUED',updated_at=? WHERE status='PENDING'`, now)
	return e
}
func (s *Store) RefreshQueue(ctx context.Context) error {
	now := time.Now().UnixNano()
	_, e := s.db.ExecContext(ctx, `UPDATE jobs SET status='QUEUED',updated_at=? WHERE status='RETRY_WAIT' AND lease_until<=?`, now, now)
	return e
}
func (s *Store) RecoverExpired(ctx context.Context) error {
	now := time.Now().UnixNano()
	_, e := s.db.ExecContext(ctx, `UPDATE jobs SET status=CASE WHEN attempt<=max_retries THEN 'RETRY_WAIT' ELSE 'FAILED' END,error='lease expired',lease_until=?,updated_at=? WHERE status IN ('LEASED','RUNNING') AND lease_until<?`, now, now, now)
	if e != nil {
		return e
	}
	return s.RefreshQueue(ctx)
}
func (s *Store) Cancel(ctx context.Context, id string) error {
	r, e := s.db.ExecContext(ctx, `UPDATE jobs SET status='CANCELLED',updated_at=? WHERE id=? AND status IN ('PENDING','QUEUED','LEASED','RUNNING','RETRY_WAIT')`, time.Now().UnixNano(), id)
	if e != nil {
		return e
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return jobs.ErrInvalidTransition
	}
	return nil
}
