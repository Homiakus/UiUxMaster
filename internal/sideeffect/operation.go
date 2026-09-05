package sideeffect

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidOperation  = errors.New("sideeffect: invalid operation")
	ErrOperationConflict = errors.New("sideeffect: operation id reused with different payload")
	ErrCASConflict       = errors.New("sideeffect: compare-and-swap conflict")
)

type RetryClass string

const (
	RetrySafe       RetryClass = "safe_retry"
	RetryIdempotent RetryClass = "retry_with_idempotency_key"
	RetryHuman      RetryClass = "non_retryable_human_review"
)

// Operation is the stable identity of one externally visible semantic effect.
// Attempt numbers are deliberately excluded: a retry/restart must regenerate the
// same ID for the same run/activity/iteration/payload.
type Operation struct {
	RunID         string     `json:"run_id"`
	Activity      string     `json:"activity"`
	Iteration     int        `json:"iteration,omitempty"`
	Kind          string     `json:"kind"`
	PayloadDigest string     `json:"payload_digest"`
	RetryClass    RetryClass `json:"retry_class"`
}

func (o Operation) Validate() error {
	if strings.TrimSpace(o.RunID) == "" || strings.TrimSpace(o.Activity) == "" || strings.TrimSpace(o.Kind) == "" || strings.TrimSpace(o.PayloadDigest) == "" {
		return ErrInvalidOperation
	}
	return nil
}

func (o Operation) ID() (string, error) {
	if err := o.Validate(); err != nil {
		return "", err
	}
	canonical := fmt.Sprintf("%s\x00%s\x00%d\x00%s\x00%s", strings.TrimSpace(o.RunID), strings.TrimSpace(o.Activity), o.Iteration, strings.TrimSpace(o.Kind), strings.TrimSpace(o.PayloadDigest))
	sum := sha256.Sum256([]byte(canonical))
	return "op_" + hex.EncodeToString(sum[:16]), nil
}

// Receipt is durable/auditable proof that the target accepted an operation.
// Reused=true means replay found the original semantic outcome instead of
// performing the side effect again.
type Receipt struct {
	OperationID   string     `json:"operation_id"`
	Kind          string     `json:"kind"`
	PayloadDigest string     `json:"payload_digest"`
	ResultDigest  string     `json:"result_digest,omitempty"`
	RetryClass    RetryClass `json:"retry_class"`
	Applied       bool       `json:"applied"`
	Reused        bool       `json:"reused"`
}

func DigestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func DigestJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return DigestBytes(raw), nil
}
