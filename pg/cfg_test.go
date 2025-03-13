package pg_test

import (
	"testing"

	"github.com/oliveiracleidson/go-lockbox/pg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostgresLockerConfig_WithDefaults(t *testing.T) {
	config := pg.NewPostgresLockerConfig()

	assert.Equal(t, "public", config.MigrationSchema)
	assert.Equal(t, "locker_migrations", config.MigrationTableName)
	assert.Equal(t, "public", config.LockSchema)
	assert.Equal(t, "locker_locks", config.LockTableName)
	assert.Equal(t, true, config.CreateSchemasIfNotExists)
}

func TestPostgresLockerConfig_Validate(t *testing.T) {
	t.Run("Valid config should pass validation", func(t *testing.T) {
		config := pg.NewPostgresLockerConfig()
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("Invalid config should return errors", func(t *testing.T) {
		config := &pg.PostgresLockerConfig{} // Config vazia
		err := config.Validate()
		require.Error(t, err)
		assert.ErrorAs(t, err, &pg.ErrInvalidConfig)
		assert.Contains(t, err.Error(), "MigrationSchema is required")
		assert.Contains(t, err.Error(), "MigrationTableName is required")
		assert.Contains(t, err.Error(), "LockSchema is required")
		assert.Contains(t, err.Error(), "LockTableName is required")
		assert.NotContains(t, err.Error(), "LockTableName and MigrationTableName must be different")
	})
}

func TestPostgresLockerConfig_Validate_DifferentTableNames(t *testing.T) {
	config := pg.NewPostgresLockerConfig()
	config.MigrationTableName = "migrations"
	config.LockTableName = "migrations"

	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LockTableName and MigrationTableName must be different")
}

func TestPostgresLockerConfig_Setters(t *testing.T) {
	config := pg.NewPostgresLockerConfig()

	config.SetMigrationSchema("custom_schema")
	config.SetMigrationTableName("custom_migrations")
	config.SetLockSchema("custom_lock_schema")
	config.SetLockTableName("custom_lock_table")
	config.SetCreateSchemasIfNotExists(false)

	assert.Equal(t, "custom_schema", config.MigrationSchema)
	assert.Equal(t, "custom_migrations", config.MigrationTableName)
	assert.Equal(t, "custom_lock_schema", config.LockSchema)
	assert.Equal(t, "custom_lock_table", config.LockTableName)
	assert.Equal(t, false, config.CreateSchemasIfNotExists)
}
