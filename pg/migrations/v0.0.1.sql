CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- Principal table for storing distributed locks
CREATE TABLE "{{ LockSchema }}"."{{ LockTable }}" (
    key TEXT PRIMARY KEY
        CHECK (
            key ~ '^[a-zA-Z0-9_-]+$' AND 
            LENGTH(key) BETWEEN 1 AND 256
        ),
    lease_id TEXT NOT NULL,
    valid_until TIMESTAMPTZ NOT NULL,
    server_nonce TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- Auxiliary function for atomic lock acquisition
CREATE OR REPLACE FUNCTION "{{ LockSchema }}".try_acquire_lock(
    _key TEXT,
    _lease_id TEXT,
    _ttl_ms BIGINT,
    _nonce TEXT,
    _metadata JSONB
) RETURNS TABLE(
    result_acquired BOOLEAN,
    result_valid_until TIMESTAMPTZ
) AS $$
BEGIN
    -- Security checks
    IF LENGTH(_key) > 256 OR _key !~ '^[a-zA-Z0-9_-]+$' THEN
        RAISE EXCEPTION 'Invalid key format' USING ERRCODE = '22023';
    END IF;

    -- Is added 10 milliseconds to the expiration time
    -- because the network latency can cause the lock to expire before the client receives the response
    INSERT INTO "{{ LockSchema }}"."{{ LockTable }}" 
    VALUES (
        _key,
        _lease_id,
        NOW() + (_ttl_ms * INTERVAL '1 millisecond') + (10 * INTERVAL '1 millisecond'),
        _nonce,
        _metadata,
        NOW(),
        NOW()
    )
    ON CONFLICT (key) DO UPDATE SET
        lease_id = EXCLUDED.lease_id,
        valid_until = EXCLUDED.valid_until,
        server_nonce = EXCLUDED.server_nonce,
        metadata = EXCLUDED.metadata,
        updated_at = NOW()
    WHERE "{{ LockSchema }}"."{{ LockTable }}".valid_until <= NOW()
    RETURNING TRUE, valid_until INTO result_acquired, result_valid_until;  -- Store the result in the output variables
    
    -- Return the result of the operation if the lock was acquired
    RETURN QUERY SELECT COALESCE(result_acquired, FALSE), result_valid_until;
EXCEPTION
    WHEN unique_violation THEN
        RETURN QUERY SELECT FALSE, NULL;
END;
$$ LANGUAGE plpgsql VOLATILE;

-- View for health monitoring
CREATE VIEW "{{ LockSchema }}".lock_health AS
SELECT
    COUNT(*) FILTER (WHERE valid_until > NOW()) AS active_locks,
    COUNT(*) FILTER (WHERE valid_until <= NOW()) AS expired_locks,
    MIN(valid_until - NOW()) FILTER (WHERE valid_until > NOW()) AS oldest_lock_ttl,
    AVG(EXTRACT(EPOCH FROM (valid_until - created_at))) AS avg_ttl_seconds
FROM "{{ LockSchema }}"."{{ LockTable }}";