package pg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/oliveiracleidson/go-lockbox/core"
)

// i.pool = pgxpool.Pool

var (
	refreshLockSQL = `
	UPDATE "%s"."%s"
	SET
			valid_until = NOW() + ($ttl * INTERVAL '1 millisecond'),
			server_nonce = $new_nonce,
			updated_at = NOW()
	WHERE
			key = $1 AND
			lease_id = $2 AND
			server_nonce = $3 AND
			valid_until > NOW() - ($ttl * 0.15 * INTERVAL '1 millisecond');
	RETURNING valid_until;`
)

func (i *PostgresLockAdapter) Refresh(ctx context.Context, token *core.LockToken, newTTL time.Duration) (*core.LockToken, error) {

	row := i.pool.QueryRow(ctx,
		fmt.Sprintf(refreshLockSQL, i.Cfg.LockSchema, i.Cfg.LockTableName),
		token.Key, token.LeaseID, token.ServerNonce,
	)

	var valid_until time.Time
	err := row.Scan(&valid_until)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core.ErrRefreshTooLate
		}
		return nil, err
	}
	token.ValidUntil = valid_until

	return token, nil
}
