package pg

import (
	"fmt"
	"strings"
)

type PostgresLockerConfig struct {
	MigrationSchema          string
	MigrationTableName       string
	LockSchema               string
	LockTableName            string
	CreateSchemasIfNotExists bool
}

// NewPostgresLockerConfig creates a new instance of PostgresLockerConfig
// with default values.
//
// CreateSchemasIfNotExists is set to true by default.
func NewPostgresLockerConfig() *PostgresLockerConfig {
	r := &PostgresLockerConfig{
		CreateSchemasIfNotExists: true,
	}
	return r.WithDefaults()
}

func (p *PostgresLockerConfig) Validate() error {
	msgs := []string{}
	if p.MigrationSchema == "" {
		msgs = append(msgs, "MigrationSchema is required")
	}
	if p.MigrationTableName == "" {
		msgs = append(msgs, "MigrationTableName is required")
	}
	if p.LockSchema == "" {
		msgs = append(msgs, "LockSchema is required")
	}
	if p.LockTableName == "" {
		msgs = append(msgs, "LockTableName is required")
	}

	if p.LockTableName != "" && p.LockTableName == p.MigrationTableName {
		msgs = append(msgs, "LockTableName and MigrationTableName must be different")
	}

	if len(msgs) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidConfig, strings.Join(msgs, ", "))
	}

	return nil
}

// WithDefaults sets default values for missing fields
// if they are not provided.
//
// Returns the same instance
// Defaults:
//
// - MigrationSchema: public
//
// - MigrationTableName: locker_migrations
//
// - LockSchema: public
//
// - LockTableName: locker_locks
func (p *PostgresLockerConfig) WithDefaults() *PostgresLockerConfig {
	if p.MigrationSchema == "" {
		p.MigrationSchema = "public"
	}
	if p.MigrationTableName == "" {
		p.MigrationTableName = "locker_migrations"
	}
	if p.LockSchema == "" {
		p.LockSchema = "public"
	}
	if p.LockTableName == "" {
		p.LockTableName = "locker_locks"
	}

	return p
}

// SetMigrationSchema sets the MigrationSchema field.
//
// This method exists to allow functional options to set the field
// in fluent style.
func (p *PostgresLockerConfig) SetMigrationSchema(v string) *PostgresLockerConfig {
	p.MigrationSchema = v
	return p
}

// SetMigrationTableName sets the MigrationTableName field.
//
// This method exists to allow functional options to set the field
// in fluent style.
func (p *PostgresLockerConfig) SetMigrationTableName(v string) *PostgresLockerConfig {
	p.MigrationTableName = v
	return p
}

// SetLockSchema sets the LockSchema field.
//
// This method exists to allow functional options to set the field
// in fluent style.
func (p *PostgresLockerConfig) SetLockSchema(v string) *PostgresLockerConfig {
	p.LockSchema = v
	return p
}

// SetLockTableName sets the LockTableName field.
//
// This method exists to allow functional options to set the field
// in fluent style.
func (p *PostgresLockerConfig) SetLockTableName(v string) *PostgresLockerConfig {
	p.LockTableName = v
	return p
}

// SetCreateSchemasIfNotExists sets the LockTableName field.
//
// This method exists to allow functional options to set the field
// in fluent style.
func (p *PostgresLockerConfig) SetCreateSchemasIfNotExists(v bool) *PostgresLockerConfig {
	p.CreateSchemasIfNotExists = v
	return p
}
