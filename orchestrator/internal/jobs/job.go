package jobs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type Status string

const (
	Pending   Status = "PENDING"
	Queued    Status = "QUEUED"
	Leased    Status = "LEASED"
	Running   Status = "RUNNING"
	RetryWait Status = "RETRY_WAIT"
	Succeeded Status = "SUCCEEDED"
	Failed    Status = "FAILED"
	Cancelled Status = "CANCELLED"
)

type Job struct {
	ID             string          `json:"job_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	PayloadHash    string          `json:"payload_hash"`
	Type           string          `json:"type"`
	Config         json.RawMessage `json:"config"`
	Priority       int             `json:"priority"`
	Status         Status          `json:"status"`
	Attempt        int             `json:"attempt"`
	MaxRetries     int             `json:"max_retries"`
	WorkerID       string          `json:"worker_id,omitempty"`
	LeaseToken     string          `json:"lease_token,omitempty"`
	LeaseUntil     time.Time       `json:"lease_until,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Error          string          `json:"error,omitempty"`
	Artifacts      []Artifact      `json:"artifacts,omitempty"`
}
type Artifact struct {
	Name   string `json:"name"`
	URI    string `json:"uri"`
	SHA256 string `json:"sha256"`
}

var ErrConflict = errors.New("idempotency key reused with different payload")
var ErrStaleLease = errors.New("stale or invalid lease")
var ErrInvalidTransition = errors.New("invalid job state transition")

func HashPayload(typ string, config json.RawMessage) string {
	b, _ := json.Marshal(struct {
		Type   string          `json:"type"`
		Config json.RawMessage `json:"config"`
	}{typ, config})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func CanTransition(from, to Status) bool {
	switch from {
	case Pending:
		return to == Queued || to == Cancelled
	case Queued:
		return to == Leased || to == Cancelled
	case Leased:
		return to == Running || to == RetryWait || to == Failed || to == Cancelled
	case Running:
		return to == Succeeded || to == RetryWait || to == Failed || to == Cancelled
	case RetryWait:
		return to == Queued || to == Cancelled
	default:
		return false
	}
}
func Transition(j *Job, to Status) error {
	if !CanTransition(j.Status, to) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, j.Status, to)
	}
	j.Status, j.UpdatedAt = to, time.Now().UTC()
	return nil
}
