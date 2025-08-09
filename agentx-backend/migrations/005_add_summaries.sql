-- Add importance scoring to messages
ALTER TABLE messages 
ADD COLUMN IF NOT EXISTS importance FLOAT DEFAULT 0.5,
ADD COLUMN IF NOT EXISTS importance_flags JSONB;

-- Create table for session summaries
CREATE TABLE IF NOT EXISTS session_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Summary details
    summary_text TEXT NOT NULL,
    message_count INTEGER NOT NULL, -- Number of messages summarized
    start_message_id UUID REFERENCES messages(id),
    end_message_id UUID REFERENCES messages(id),
    
    -- Metadata
    tokens_saved INTEGER, -- Approximate tokens saved by using summary
    model_used VARCHAR(100), -- Model used to generate summary
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index for fast lookups
CREATE INDEX idx_session_summaries_session_id ON session_summaries(session_id);
CREATE INDEX idx_session_summaries_created_at ON session_summaries(created_at DESC);

-- Create trigger to update updated_at
CREATE TRIGGER update_session_summaries_updated_at
    BEFORE UPDATE ON session_summaries
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();