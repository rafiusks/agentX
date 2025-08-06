-- Add user_id to existing tables

-- Update sessions table
ALTER TABLE sessions 
    ADD COLUMN user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE;

-- Update provider_connections table
ALTER TABLE provider_connections 
    ADD COLUMN user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE;

-- Update default_connections table  
ALTER TABLE default_connections 
    ADD COLUMN user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE;

-- Update configs table (if needed for user-specific configs)
ALTER TABLE configs 
    ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;

-- Add indexes for performance
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_provider_connections_user_id ON provider_connections(user_id);
CREATE INDEX idx_default_connections_user_id ON default_connections(user_id);
CREATE INDEX idx_configs_user_id ON configs(user_id) WHERE user_id IS NOT NULL;

-- Add composite unique constraints where needed
ALTER TABLE default_connections 
    ADD CONSTRAINT unique_user_provider_default 
    UNIQUE (user_id, provider_id);

-- Make sure provider connections are unique per user and name
ALTER TABLE provider_connections 
    ADD CONSTRAINT unique_user_connection_name 
    UNIQUE (user_id, name);