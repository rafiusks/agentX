-- Create provider_configs table
CREATE TABLE IF NOT EXISTS provider_configs (
    id VARCHAR(100) PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    base_url VARCHAR(500),
    api_key TEXT, -- Encrypted in application layer
    models JSONB DEFAULT '[]'::jsonb,
    default_model VARCHAR(255),
    extra JSONB DEFAULT '{}'::jsonb,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_provider_configs_type ON provider_configs(type);
CREATE INDEX idx_provider_configs_active ON provider_configs(is_active);

-- Add default providers (without API keys)
INSERT INTO provider_configs (id, type, name, models, default_model) VALUES 
    ('openai', 'openai', 'OpenAI', '["gpt-4", "gpt-3.5-turbo"]'::jsonb, 'gpt-3.5-turbo'),
    ('anthropic', 'anthropic', 'Anthropic', '["claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307"]'::jsonb, 'claude-3-sonnet-20240229'),
    ('ollama', 'openai-compatible', 'Ollama', '[]'::jsonb, ''),
    ('lm-studio', 'openai-compatible', 'LM Studio', '[]'::jsonb, '')
ON CONFLICT (id) DO NOTHING;

-- Update base URLs for local providers
UPDATE provider_configs SET base_url = 'http://localhost:11434' WHERE id = 'ollama';
UPDATE provider_configs SET base_url = 'http://localhost:1234' WHERE id = 'lm-studio';

-- Create trigger to update updated_at
CREATE TRIGGER update_provider_configs_updated_at BEFORE UPDATE ON provider_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create app_settings table for global settings
CREATE TABLE IF NOT EXISTS app_settings (
    key VARCHAR(255) PRIMARY KEY,
    value JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add default settings
INSERT INTO app_settings (key, value) VALUES 
    ('default_provider', '"openai"'::jsonb),
    ('default_model', '""'::jsonb)
ON CONFLICT (key) DO NOTHING;

-- Create trigger for app_settings
CREATE TRIGGER update_app_settings_updated_at BEFORE UPDATE ON app_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();