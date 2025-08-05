-- Down migration for provider connections
DROP TABLE IF EXISTS default_connections;
DROP TABLE IF EXISTS provider_connections;
DROP FUNCTION IF EXISTS update_provider_connections_updated_at();