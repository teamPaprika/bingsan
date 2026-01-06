-- Add columns for API key rotation
ALTER TABLE api_keys ADD COLUMN previous_key_hash TEXT;
ALTER TABLE api_keys ADD COLUMN rotated_at TIMESTAMPTZ;

-- Index for faster lookup of previous key (used during rotation grace period)
CREATE INDEX idx_api_keys_previous_key_hash ON api_keys(previous_key_hash) WHERE previous_key_hash IS NOT NULL;

COMMENT ON COLUMN api_keys.previous_key_hash IS 'Hash of the previous key, valid during rotation grace period';
COMMENT ON COLUMN api_keys.rotated_at IS 'Timestamp when the key was last rotated';
