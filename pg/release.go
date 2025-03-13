package pg

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/oliveiracleidson/go-lockbox/core"
)

// i.pool = pgxpool.Pool

var (
	releaseLockSQL = `
	DELETE FROM "%s"."%s"
	WHERE
  	key = $1
		AND lease_id = $2 
		AND server_nonce = $3;`
)

func (i *PostgresLockAdapter) Release(ctx context.Context, token *core.LockToken) error {

	r, err := i.pool.Exec(ctx,
		fmt.Sprintf(releaseLockSQL, i.Cfg.LockSchema, i.Cfg.LockTableName),
		token.Key, token.LeaseID, token.ServerNonce,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.ErrLockOwnershipMismatch
		} else {
			return err
		}
	}

	if r.RowsAffected() == 0 {
		return core.ErrLockOwnershipMismatch
	}

	return nil
}
