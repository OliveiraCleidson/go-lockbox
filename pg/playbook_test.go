package pg_test

import (
	"context"
	"testing"
	"time"

	"github.com/oliveiracleidson/go-lockbox/core"
	"github.com/stretchr/testify/require"
)

func TestNewPostgresLockerMigration_Migration(t *testing.T) {
	t.Run(("when public schema exists, then return status"), func(t *testing.T) {
		res, err := adapter.GetSchemaStatus(context.Background())
		require.NoError(t, err)
		require.NotNil(t, res)
		require.True(t, res.MigrationSchemaExists)
		require.False(t, res.MigrationTableExists)
		require.True(t, res.LockSchemaExists)
		require.False(t, res.LockTableExists)
	})

	t.Run("given schemas that not exists, then create schemas and migration table", func(t *testing.T) {
		adapter.Cfg.MigrationSchema = "locker"
		adapter.Cfg.MigrationTableName = "migrations"
		adapter.Cfg.LockSchema = "locker"
		adapter.Cfg.LockTableName = "locks"

		res, err := adapter.GetSchemaStatus(context.Background())
		require.NoError(t, err)
		require.NotNil(t, res)
		require.False(t, res.MigrationSchemaExists)
		require.False(t, res.MigrationTableExists)
		require.False(t, res.LockSchemaExists)
		require.False(t, res.LockTableExists)

		err = adapter.PrepareDbForMigrations(context.Background())
		require.NoError(t, err)

		res, err = adapter.GetSchemaStatus(context.Background())
		require.NoError(t, err)
		require.NotNil(t, res)
		require.True(t, res.MigrationSchemaExists)
		require.True(t, res.MigrationTableExists)
		require.True(t, res.LockSchemaExists)
		require.False(t, res.LockTableExists)
	})

	t.Run("when run migrations, then create lock table", func(t *testing.T) {
		res, err := adapter.GetSchemaStatus(context.Background())
		require.NoError(t, err)
		require.NotNil(t, res)
		require.False(t, res.LockTableExists)

		err = adapter.RunMigrations(context.Background())
		require.NoError(t, err)

		res, err = adapter.GetSchemaStatus(context.Background())
		require.NoError(t, err)
		require.NotNil(t, res)
		require.True(t, res.LockTableExists)
	})

	t.Run("given a key with metadata and lock is not acquired by others, then create lock", func(t *testing.T) {
		res, err := adapter.Acquire(
			context.Background(),
			"key",
			core.LockOptions{
				TTL: 10 * time.Second,
				RetryStrategy: core.RetryStrategy{
					MaxRetries:    5,
					BaseDelay:     100 * time.Millisecond,
					MaxDelay:      10 * time.Second,
					JitterFactor:  0.2,
					BackoffFactor: 2,
				},
				Metadata: map[string]string{
					"owner": "test",
				},
				RequestTimeout: 5 * time.Second,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "key", res.Key)
		require.NotEmpty(t, res.LeaseID)
		require.NotEmpty(t, res.ServerNonce)
		require.NotEmpty(t, res.ValidUntil)
	})

	t.Run("given a key without metadata and lock is not acquired by others, when acquire lock, then create lock", func(t *testing.T) {
		res, err := adapter.Acquire(
			context.Background(),
			"key-without-metadata",
			core.LockOptions{
				TTL: 10 * time.Second,
				RetryStrategy: core.RetryStrategy{
					MaxRetries:    5,
					BaseDelay:     100 * time.Millisecond,
					MaxDelay:      10 * time.Second,
					JitterFactor:  0.2,
					BackoffFactor: 2,
				},
				Metadata:       nil,
				RequestTimeout: 5 * time.Second,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "key-without-metadata", res.Key)
		require.NotEmpty(t, res.LeaseID)
		require.NotEmpty(t, res.ServerNonce)
		require.NotEmpty(t, res.ValidUntil)
	})

	t.Run("given a key and lock is acquired by others, when acquire lock, then returns error", func(t *testing.T) {
		res, err := adapter.Acquire(
			context.Background(),
			"key-lock",
			core.LockOptions{
				TTL: 10 * time.Second,
				RetryStrategy: core.RetryStrategy{
					MaxRetries:    5,
					BaseDelay:     100 * time.Millisecond,
					MaxDelay:      10 * time.Second,
					JitterFactor:  0.2,
					BackoffFactor: 2,
				},
				Metadata:       nil,
				RequestTimeout: 5 * time.Second,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, res)

		res, err = adapter.Acquire(
			context.Background(),
			"key-lock",
			core.LockOptions{
				TTL: 10 * time.Second,
				RetryStrategy: core.RetryStrategy{
					MaxRetries:    5,
					BaseDelay:     100 * time.Millisecond,
					MaxDelay:      10 * time.Second,
					JitterFactor:  0.2,
					BackoffFactor: 2,
				},
				Metadata:       nil,
				RequestTimeout: 5 * time.Second,
			},
		)
		require.Error(t, err)
		require.Nil(t, res)
		require.ErrorAs(t, err, &core.ErrLockAcquisitionFailed)
	})

	t.Run("given a key released, when try to acquire the key, then acquire with success", func(t *testing.T) {
		firstLock, err := adapter.Acquire(
			context.Background(),
			"key-lock-will-be-released",
			core.LockOptions{
				TTL: core.MaxLockTTL,
				RetryStrategy: core.RetryStrategy{
					MaxRetries:    5,
					BaseDelay:     100 * time.Millisecond,
					MaxDelay:      10 * time.Second,
					JitterFactor:  0.2,
					BackoffFactor: 2,
				},
				Metadata:       nil,
				RequestTimeout: 5 * time.Second,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, firstLock)

		res, err := adapter.Acquire(
			context.Background(),
			"key-lock-will-be-released",
			core.LockOptions{
				TTL: 10 * time.Second,
				RetryStrategy: core.RetryStrategy{
					MaxRetries:    5,
					BaseDelay:     100 * time.Millisecond,
					MaxDelay:      10 * time.Second,
					JitterFactor:  0.2,
					BackoffFactor: 2,
				},
				Metadata:       nil,
				RequestTimeout: 5 * time.Second,
			},
		)
		require.Error(t, err)
		require.Nil(t, res)
		require.ErrorAs(t, err, &core.ErrLockAcquisitionFailed)

		err = adapter.Release(context.Background(), firstLock)
		require.NoError(t, err)

		res, err = adapter.Acquire(
			context.Background(),
			"key-lock-will-be-released",
			core.LockOptions{
				TTL: core.MaxLockTTL,
				RetryStrategy: core.RetryStrategy{
					MaxRetries:    5,
					BaseDelay:     100 * time.Millisecond,
					MaxDelay:      10 * time.Second,
					JitterFactor:  0.2,
					BackoffFactor: 2,
				},
				Metadata:       nil,
				RequestTimeout: 5 * time.Second,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotEqual(t, firstLock.LeaseID, res.LeaseID)
		require.NotEqual(t, firstLock.ServerNonce, res.ServerNonce)
	})
}
