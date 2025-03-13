package pg

import (
	"context"
	"embed"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

type migrationData struct {
	Version     string
	FileName    string
	Transaction bool
}

// Migrations File
var (
	//go:embed migrations/*.sql
	migrationsEmbed embed.FS
	migrationsData  = []migrationData{
		{Version: "v0.0.1", FileName: "migrations/v0.0.1.sql", Transaction: true},
		{Version: "v0.0.1-indexes", FileName: "migrations/v0.0.1-indexes.sql", Transaction: false},
	}
)

type schemaStatus struct {
	MigrationSchemaExists bool
	MigrationTableExists  bool
	LockSchemaExists      bool
	LockTableExists       bool
}

// Queries
var (
	schemaExistsQuery = `
	SELECT EXISTS (
		SELECT 
			schema_name 
		FROM information_schema.schemata 
		WHERE schema_name = $1
	) as exists;`
	tableExistsQuery = `
	SELECT EXISTS (
		SELECT 
			table_name 
		FROM information_schema.tables 
		WHERE table_schema = $1 
		AND table_name = $2
	);
	`
)

// Returns the status of existance of the migration and lock schemas and tables
func (i *PostgresLockAdapter) GetSchemaStatus(ctx context.Context) (*schemaStatus, error) {
	status := &schemaStatus{
		MigrationSchemaExists: false,
		MigrationTableExists:  false,
		LockSchemaExists:      false,
		LockTableExists:       false,
	}

	rows := i.pool.QueryRow(
		ctx,
		schemaExistsQuery,
		i.Cfg.MigrationSchema,
	)
	err := rows.Scan(&status.MigrationSchemaExists)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	if i.Cfg.LockSchema == i.Cfg.MigrationSchema {
		status.LockSchemaExists = status.MigrationSchemaExists
	}

	rows = i.pool.QueryRow(
		ctx,
		tableExistsQuery,
		i.Cfg.MigrationSchema,
		i.Cfg.MigrationTableName,
	)
	err = rows.Scan(&status.MigrationTableExists)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	if i.Cfg.LockSchema != i.Cfg.MigrationSchema {
		rows = i.pool.QueryRow(
			ctx,
			schemaExistsQuery,
			i.Cfg.LockSchema,
		)
		err = rows.Scan(&status.LockSchemaExists)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
		}
	}

	rows = i.pool.QueryRow(
		ctx,
		tableExistsQuery,
		i.Cfg.LockSchema,
		i.Cfg.LockTableName,
	)
	err = rows.Scan(&status.LockTableExists)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	return status, nil
}

func (i *PostgresLockAdapter) PrepareDbForMigrations(ctx context.Context) error {
	if !i.Cfg.CreateSchemasIfNotExists {
		return nil
	}

	err := i.createMigrationSchema(ctx)
	if err != nil {
		return err
	}
	err = i.createLockSchema(ctx)
	if err != nil {
		return err
	}

	err = i.createMigrationTable(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (i *PostgresLockAdapter) RunMigrations(ctx context.Context) error {
	for _, migration := range migrationsData {
		err := i.runMigration(ctx, migration)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *PostgresLockAdapter) runMigration(ctx context.Context, migration migrationData) error {
	if migration.Transaction {
		return i.runMigrationTransaction(ctx, migration)
	}

	migrationData, err := migrationsEmbed.ReadFile(migration.FileName)
	if err != nil {
		return err
	}

	sql := string(migrationData)
	sql = strings.ReplaceAll(sql, "{{ LockSchema }}", i.Cfg.LockSchema)
	sql = strings.ReplaceAll(sql, "{{ LockTable }}", i.Cfg.LockTableName)

	conn, err := i.pool.Acquire(ctx)
	if err != nil {
		return err
	}

	defer conn.Release()

	// split by ;
	queries := strings.Split(sql, ";")
	for _, query := range queries {
		rows := conn.QueryRow(ctx, query)
		err = rows.Scan()
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}

	_, err = conn.Exec(
		ctx,
		"INSERT INTO "+i.Cfg.MigrationSchema+"."+i.Cfg.MigrationTableName+" (version) VALUES ($1)",
		migration.Version,
	)
	if err != nil {
		return err
	}

	return nil
}

func (i *PostgresLockAdapter) runMigrationTransaction(ctx context.Context, migration migrationData) error {
	tx, err := i.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	migrationData, err := migrationsEmbed.ReadFile(migration.FileName)
	if err != nil {
		return err
	}

	sql := string(migrationData)
	sql = strings.ReplaceAll(sql, "{{ LockSchema }}", i.Cfg.LockSchema)
	sql = strings.ReplaceAll(sql, "{{ LockTable }}", i.Cfg.LockTableName)
	_, err = tx.Exec(ctx, sql)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		ctx,
		"INSERT INTO "+i.Cfg.MigrationSchema+"."+i.Cfg.MigrationTableName+" (version) VALUES ($1)",
		migration.Version,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (i *PostgresLockAdapter) createMigrationSchema(ctx context.Context) error {
	_, err := i.pool.Exec(
		ctx,
		"CREATE SCHEMA IF NOT EXISTS "+i.Cfg.MigrationSchema,
	)
	return err
}

func (i *PostgresLockAdapter) createLockSchema(ctx context.Context) error {
	_, err := i.pool.Exec(
		ctx,
		"CREATE SCHEMA IF NOT EXISTS "+i.Cfg.LockSchema,
	)
	return err
}

func (i *PostgresLockAdapter) createMigrationTable(ctx context.Context) error {
	_, err := i.pool.Exec(
		ctx,
		`CREATE TABLE IF NOT EXISTS `+i.Cfg.MigrationSchema+`.`+i.Cfg.MigrationTableName+` (
			id SERIAL PRIMARY KEY,
			version varchar(50) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
	)
	return err
}
