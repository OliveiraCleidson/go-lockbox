package pg

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/oliveiracleidson/go-lockbox/core"
)

// i.pool = pgxpool.Pool

func (i *PostgresLockAdapter) Acquire(ctx context.Context, key string, opts core.LockOptions) (*core.LockToken, error) {
	if err := core.ValidateKey(key); err != nil {
		return nil, err
	}
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	leaseID := uuid.NewString()
	nonce := uuid.NewString()
	metadata, err := json.Marshal(opts.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var lockToken *core.LockToken

	for attempt := 0; attempt <= opts.RetryStrategy.MaxRetries; attempt++ {
		txCtx, cancel := context.WithTimeout(ctx, opts.RequestTimeout)
		defer cancel()

		row := i.pool.QueryRow(txCtx,
			fmt.Sprintf(`SELECT * FROM "%s".try_acquire_lock($1, $2, $3, $4, $5)`, i.Cfg.LockSchema),
			key, leaseID, opts.TTL.Milliseconds(), nonce, metadata,
		)

		var acquired bool
		var validUntil time.Time
		err := row.Scan(&acquired, &validUntil)
		if err == nil && acquired {
			lockToken = &core.LockToken{
				Key:         key,
				LeaseID:     leaseID,
				ValidUntil:  validUntil,
				ServerNonce: nonce,
			}
			return lockToken, nil
		}

		// Se o erro for relacionado a contenção de lock, tentamos novamente com backoff
		if err == nil && !acquired {
			time.Sleep(core.CalculateBackoff(opts.RetryStrategy, attempt))
			continue
		}

		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	return nil, core.ErrLockAcquisitionFailed
}
