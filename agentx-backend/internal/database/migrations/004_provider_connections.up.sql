-- Create provider_connections table to support multiple named connections per provider
CREATE TABLE IF NOT EXISTS provider_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id VARCHAR(50) NOT NULL, -- e.g., 'openai', 'anthropic', 'ollama'
    name VARCHAR(255) NOT NULL, -- User-defined connection name
    enabled BOOLEAN DEFAULT true,
    config JSONB NOT NULL DEFAULT '{}', -- Provider-specific configuration
    metadata JSONB DEFAULT '{}', -- Additional metadata (last_tested, etc.)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient querying
CREATE INDEX idx_provider_connections_provider_id ON provider_connections(provider_id);
CREATE INDEX idx_provider_connections_enabled ON provider_connections(enabled);

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_provider_connections_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER provider_connections_updated_at
    BEFORE UPDATE ON provider_connections
    FOR EACH ROW
    EXECUTE FUNCTION update_provider_connections_updated_at();

-- Migrate existing provider_configs data if it exists
INSERT INTO provider_connections (provider_id, name, enabled, config)
SELECT 
    provider_id,
    name || ' (Migrated)' as name,
    true as enabled,
    jsonb_build_object(
        'api_key', api_key,
        'base_url', base_url,
        'models', models,
        'default_model', default_model
    ) as config
FROM provider_configs
WHERE EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'provider_configs');

-- Add default connections table for storing user's preferred connection per provider
CREATE TABLE IF NOT EXISTS default_connections (
    provider_id VARCHAR(50) PRIMARY KEY,
    connection_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (connection_id) REFERENCES provider_connections(id) ON DELETE CASCADE
);