// Package core provides a unified interface for distributed locks
// with support for postgres and in future multiple backends (Redis, etcd, etc.).
//
// Overview:
// This package abstracts distributed locking operations with:
// - Atomic lock acquisition
// - Secure renewal with ownership verification
// - Time-to-live (TTL) control
// - Service health monitoring
// - Integrated metrics
//
// Common Use Cases:
// - Distributed process coordination
// - Race condition prevention
// - Shared resource access control
// - Implementation of distributed queues
package core

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"
)

// Package-specific errors
var (
	// Lock acquisition failed after retries
	ErrLockAcquisitionFailed = errors.New("lock acquisition failed")

	// Attempt to operate on a lock not owned by the caller
	ErrLockOwnershipMismatch = errors.New("lock ownership mismatch")

	// Specified TTL is out of allowed range
	ErrInvalidTTL = errors.New("invalid TTL duration (1ms-10m)")

	// Operation attempted on a closed adapter
	ErrAdapterClosed = errors.New("lock adapter closed")

	// Contention limit exceeded
	ErrLockContention = errors.New("lock contention limit exceeded")

	// Invalid key format
	ErrInvalidKeyFormat = errors.New("invalid key format (max 256 chars, [a-zA-Z0-9_-])")

	// Renewal beyond the safe margin
	ErrRefreshTooLate = errors.New("lock refresh beyond safety margin")

	// Operation timeout
	ErrOperationTimeout = errors.New("lock operation timed out")

	// Lock not found
	ErrLockNotFound = errors.New("lock not found")
)

// Configuration constants
const (
	DefaultLockTTL        = 15 * time.Second     // Default TTL
	MinLockTTL            = 1 * time.Millisecond // Minimum TTL
	MaxLockTTL            = 10 * time.Minute     // Maximum TTL
	DefaultMaxRetries     = 5                    // Default retry attempts
	DefaultJitterFactor   = 0.3                  // Default jitter factor
	MaxClockDriftMargin   = 0.15                 // Maximum clock drift margin
	MaxKeyLength          = 256                  // Maximum key length
	DefaultRequestTimeout = 3 * time.Second      // Default timeout
)

// LockOptions defines parameters for lock acquisition
type LockOptions struct {
	TTL            time.Duration     // Lock time-to-live
	RetryStrategy  RetryStrategy     // Retry strategy
	Metadata       map[string]string // Custom metadata
	RequestTimeout time.Duration     // Per-operation timeout
}

// Validate checks LockOptions parameters
func (o *LockOptions) Validate() error {
	if o.TTL < MinLockTTL || o.TTL > MaxLockTTL {
		return fmt.Errorf("%w: %v", ErrInvalidTTL, o.TTL)
	}
	if o.RequestTimeout <= 0 {
		o.RequestTimeout = DefaultRequestTimeout
	}
	return o.RetryStrategy.Validate()
}

// RetryStrategy defines a retry policy
type RetryStrategy struct {
	MaxRetries    int           // Maximum number of attempts
	BaseDelay     time.Duration // Initial delay
	MaxDelay      time.Duration // Maximum delay
	JitterFactor  float64       // Random variation (0.0-1.0)
	BackoffFactor float64       // Exponential growth factor
}

func (r *RetryStrategy) Validate() error {
	if r.MaxRetries < 0 {
		return errors.New("max retries must be ≥ 0")
	}
	if r.JitterFactor < 0 || r.JitterFactor > 1 {
		return errors.New("jitter factor must be [0.0, 1.0]")
	}
	if r.BackoffFactor < 1 {
		return errors.New("backoff factor must be ≥ 1")
	}
	return nil
}

// LockToken represents a successfully acquired lock
type LockToken struct {
	Key         string    // Locked resource key
	LeaseID     string    // Unique lock identifier
	ValidUntil  time.Time // Absolute expiration
	ServerNonce string    // Security nonce
}

// LockAdapter main interface for distributed locks
type LockAdapter interface {
	// Acquire obtains a distributed lock
	//
	// Parameters:
	// - ctx: Context for cancellation
	// - key: Unique resource identifier
	// - opts: Configuration options
	//
	// Returns:
	// - LockToken on success
	// - ErrLockAcquisitionFailed: error if acquisition fails
	// - ErrLockContention: error if contention limit is reached
	Acquire(ctx context.Context, key string, opts LockOptions) (*LockToken, error)

	// Release frees an acquired lock
	//
	// Requirements:
	// - Validates ServerNonce to ensure ownership
	Release(ctx context.Context, token *LockToken) error

	// Refresh extends the duration of an existing lock
	//
	// Security:
	// - Checks clock drift margin
	// - Updates ServerNonce
	Refresh(ctx context.Context, token *LockToken, newTTL time.Duration) (*LockToken, error)

	// IsHeld checks lock validity and ownership
	IsHeld(ctx context.Context, token *LockToken) (bool, time.Duration, error)

	// Close safely shuts down the adapter
	Close(ctx context.Context) error

	// HealthCheck monitors service health
	HealthCheck(ctx context.Context) HealthReport
}

// HealthReport provides service health status
type HealthReport struct {
	Status     HealthStatus  // Overall state
	Latency    time.Duration // Average latency
	Throughput float64       // Operations per second
	Error      error         // Last relevant error
}

type HealthStatus int

const (
	StatusGreen HealthStatus = iota
	StatusYellow
	StatusRed
)

func ValidateKey(key string) error {
	validKeyRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,256}$`)
	if !validKeyRegex.MatchString(key) {
		return fmt.Errorf("%w: %s", ErrInvalidKeyFormat, key)
	}
	return nil
}

// Helper for calculating backoff time
func CalculateBackoff(strategy RetryStrategy, attempt int) time.Duration {
	delay := strategy.BaseDelay * time.Duration(math.Pow(
		strategy.BackoffFactor,
		float64(attempt),
	))
	if delay > strategy.MaxDelay {
		return strategy.MaxDelay
	}
	return delay
}

// Advanced Example:
//
//  // Configuration with exponential retry
//  opts := LockOptions{
//      TTL: 30 * time.Second,
//      RetryStrategy: RetryStrategy{
//          MaxRetries:    5,
//          BaseDelay:     100 * time.Millisecond,
//          MaxDelay:      10 * time.Second,
//          JitterFactor:  0.2,
//          BackoffFactor: 2,
//      },
//  }
//
//  // Acquisition with automatic retry (Internally we use CalculateBackoff)
//  for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
//      lock, err := adapter.Acquire(ctx, "resource", opts)
//      if errors.Is(err, ErrLockContention) {
//          delay := CalculateBackoff(opts.RetryStrategy, attempt)
//          time.Sleep(delay)
//          continue
//      }
//      // Handle success/error
//  }
//
// Best Practices:
// - Always validate LockToken after acquisition
// - Use conservative TTLs
// - Implement async renewal for long-running operations
// - Monitor contention metrics
// - Validate keys with ValidateKey()
//
// Observability:
// Implement LockMetrics to collect:
// - Success/failure rate
// - Acquisition time
// - Contention
// - Hold time
//
// Implementation Notes:
// - Thread-safe by design
// - Persistent connections
// - Automatic failover (backend-dependent)
// - Supports high-load systems
//
// HealthCheck Status:
// - Green: Operational
// - Yellow: Degraded (high latency/transient errors)
// - Red: Unavailable
