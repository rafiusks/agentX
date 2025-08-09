-- Context memory system for persistent knowledge across conversations
CREATE TABLE IF NOT EXISTS context_memory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id UUID,
    namespace TEXT NOT NULL DEFAULT 'default',
    key TEXT NOT NULL,
    value JSONB NOT NULL,
    -- embedding vector(1536), -- For semantic search (requires pgvector extension)
    embedding TEXT, -- Temporarily store as TEXT until pgvector is available
    importance FLOAT DEFAULT 0.5,
    access_count INTEGER DEFAULT 0,
    last_accessed TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, namespace, key)
);

-- Indexes for efficient querying
CREATE INDEX idx_context_memory_user_id ON context_memory(user_id);
CREATE INDEX idx_context_memory_project_id ON context_memory(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX idx_context_memory_namespace ON context_memory(namespace);
CREATE INDEX idx_context_memory_key ON context_memory(key);
CREATE INDEX idx_context_memory_importance ON context_memory(importance DESC);
CREATE INDEX idx_context_memory_last_accessed ON context_memory(last_accessed DESC);
CREATE INDEX idx_context_memory_expires_at ON context_memory(expires_at) WHERE expires_at IS NOT NULL;

-- Canvas artifacts for iterative work
CREATE TABLE IF NOT EXISTS canvas_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('code', 'document', 'diagram', 'data')),
    title TEXT,
    content TEXT NOT NULL,
    language TEXT, -- For code artifacts
    version INTEGER NOT NULL DEFAULT 1,
    parent_version UUID REFERENCES canvas_artifacts(id),
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for canvas artifacts
CREATE INDEX idx_canvas_artifacts_session_id ON canvas_artifacts(session_id);
CREATE INDEX idx_canvas_artifacts_user_id ON canvas_artifacts(user_id);
CREATE INDEX idx_canvas_artifacts_type ON canvas_artifacts(type);
CREATE INDEX idx_canvas_artifacts_is_active ON canvas_artifacts(is_active);
CREATE INDEX idx_canvas_artifacts_created_at ON canvas_artifacts(created_at DESC);

-- User patterns for proactive assistance
CREATE TABLE IF NOT EXISTS user_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pattern_type TEXT NOT NULL,
    pattern_data JSONB NOT NULL,
    confidence FLOAT DEFAULT 0.5,
    frequency INTEGER DEFAULT 1,
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for user patterns
CREATE INDEX idx_user_patterns_user_id ON user_patterns(user_id);
CREATE INDEX idx_user_patterns_type ON user_patterns(pattern_type);
CREATE INDEX idx_user_patterns_confidence ON user_patterns(confidence DESC);
CREATE INDEX idx_user_patterns_frequency ON user_patterns(frequency DESC);
CREATE INDEX idx_user_patterns_last_seen ON user_patterns(last_seen DESC);

-- Context memory associations (link memories to sessions/messages)
CREATE TABLE IF NOT EXISTS context_memory_refs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id UUID NOT NULL REFERENCES context_memory(id) ON DELETE CASCADE,
    session_id UUID REFERENCES sessions(id) ON DELETE CASCADE,
    message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
    relevance_score FLOAT DEFAULT 0.5,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CHECK ((session_id IS NOT NULL) OR (message_id IS NOT NULL))
);

-- Indexes for context memory references
CREATE INDEX idx_context_memory_refs_memory_id ON context_memory_refs(memory_id);
CREATE INDEX idx_context_memory_refs_session_id ON context_memory_refs(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX idx_context_memory_refs_message_id ON context_memory_refs(message_id) WHERE message_id IS NOT NULL;
CREATE INDEX idx_context_memory_refs_relevance ON context_memory_refs(relevance_score DESC);

-- Add triggers for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_context_memory_updated_at BEFORE UPDATE ON context_memory
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_canvas_artifacts_updated_at BEFORE UPDATE ON canvas_artifacts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_patterns_updated_at BEFORE UPDATE ON user_patterns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();