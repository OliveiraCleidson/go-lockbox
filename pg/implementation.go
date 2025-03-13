package pg

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oliveiracleidson/go-lockbox/core"
)

type PostgresLockAdapter struct {
	pool *pgxpool.Pool
	Cfg  *PostgresLockerConfig
}

// NewPostgresLockAdapter cria uma nova instância do adapter PostgreSQL
func NewPostgresLockAdapter(
	pool *pgxpool.Pool,
	cfg *PostgresLockerConfig,
) (*PostgresLockAdapter, error) {
	r := &PostgresLockAdapter{
		Cfg:  cfg,
		pool: pool,
	}

	return r, nil
}

// Close the pgxPool
func (p *PostgresLockAdapter) Close(ctx context.Context) error {
	p.pool.Close()
	return nil
}

// HealthCheck monitors service health.
// Throughput is the number of acquired connections and
// latency is the time taken to execute the query.
func (p *PostgresLockAdapter) HealthCheck(ctx context.Context) core.HealthReport {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	start := time.Now()
	var result int
	err := p.pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	latency := time.Since(start) // Mede apenas o tempo da query

	status := core.StatusGreen
	var errMsg string

	if err != nil || result != 1 {
		status = core.StatusRed
		if err != nil {
			errMsg = err.Error() // Registrar erro
		} else {
			errMsg = "unexpected query result"
		}
	}

	poolStats := p.pool.Stat()
	throughput := int(poolStats.AcquiredConns())

	return core.HealthReport{
		Status:     status,
		Latency:    latency,
		Throughput: float64(throughput),
		Error:      errors.New(errMsg),
	}
}
