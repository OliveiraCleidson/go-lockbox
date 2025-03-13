-- Index for automatic cleanup of expired locks
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_locks_expiration 
    ON "{{ LockSchema }}"."{{ LockTable }}" (valid_until);

-- Otimization for renewal operations
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_locks_lease 
    ON "{{ LockSchema }}"."{{ LockTable }}" (lease_id, server_nonce);