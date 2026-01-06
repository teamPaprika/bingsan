-- Remove API key rotation columns
DROP INDEX IF EXISTS idx_api_keys_previous_key_hash;
ALTER TABLE api_keys DROP COLUMN IF EXISTS previous_key_hash;
ALTER TABLE api_keys DROP COLUMN IF EXISTS rotated_at;
