-- Remove constraints
ALTER TABLE provider_connections 
    DROP CONSTRAINT IF EXISTS unique_user_connection_name;

ALTER TABLE default_connections 
    DROP CONSTRAINT IF EXISTS unique_user_provider_default;

-- Remove indexes
DROP INDEX IF EXISTS idx_configs_user_id;
DROP INDEX IF EXISTS idx_default_connections_user_id;
DROP INDEX IF EXISTS idx_provider_connections_user_id;
DROP INDEX IF EXISTS idx_sessions_user_id;

-- Remove user_id columns
ALTER TABLE configs DROP COLUMN IF EXISTS user_id;
ALTER TABLE default_connections DROP COLUMN IF EXISTS user_id;
ALTER TABLE provider_connections DROP COLUMN IF EXISTS user_id;
ALTER TABLE sessions DROP COLUMN IF EXISTS user_id;