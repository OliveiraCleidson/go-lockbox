package pg_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oliveiracleidson/go-lockbox/pg"
)

var (
	adapter *pg.PostgresLockAdapter
	pgxPool *pgxpool.Pool
	// once
	onc sync.Once
)

func TestMain(m *testing.M) {
	// Chama o setup antes dos testes
	setupImplementation()

	// Executa os testes
	code := m.Run()

	// Chama o teardown após todos os testes
	teardownImplementation()

	// Finaliza a execução dos testes
	os.Exit(code)
}

func setupImplementation() *pg.PostgresLockAdapter {
	onc.Do(func() {

		dbUrl := os.Getenv("DB_URL")
		if dbUrl == "" {
			panic("DB_URL is required for tests")
		}
		pgxConfig, err := pgxpool.ParseConfig(dbUrl)
		if err != nil {
			panic(err)
		}

		// Timeout of 5 seconds to connect to the database
		pgxConfig.ConnConfig.ConnectTimeout = 5 * time.Second
		pgxConfig.MaxConns = 50
		pgxConfig.MinConns = 1
		pool, err := pgxpool.NewWithConfig(context.Background(), pgxConfig)

		// Test connection of the database
		_, err = pool.Exec(context.Background(), "SELECT 1")
		if err != nil {
			panic(err)
		}

		pgxPool = pool

		a, err := pg.NewPostgresLockAdapter(
			pool,
			pg.NewPostgresLockerConfig(),
		)
		if err != nil {
			panic(err)
		}

		adapter = a
	})

	return adapter
}

func teardownImplementation() {
	if adapter != nil {
		adapter = nil
	}

	if pgxPool != nil {
		pgxPool.Close()
		pgxPool = nil
	}
}
