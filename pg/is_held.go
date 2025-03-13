package pg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/oliveiracleidson/go-lockbox/core"
)

var (
	isHeldLockSQL = `
	SELECT 
    	valid_until > NOW() AS is_locked,
    	EXTRACT(EPOCH FROM (valid_until - NOW())) AS remaining_ttl
	FROM "%s"."%s"
	WHERE key = $1;`
)

func (i *PostgresLockAdapter) IsHeld(ctx context.Context, token *core.LockToken) (bool, time.Duration, error) {
	row := i.pool.QueryRow(ctx,
		fmt.Sprintf(isHeldLockSQL, i.Cfg.LockSchema, i.Cfg.LockTableName),
		token.Key,
	)

	var isLocked bool
	var remainingTTL float64

	err := row.Scan(&isLocked, &remainingTTL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, 0, nil
		}
		return false, 0, err
	}

	return isLocked, time.Duration(remainingTTL) * time.Second, nil
}
