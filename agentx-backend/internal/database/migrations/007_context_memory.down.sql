-- Drop triggers
DROP TRIGGER IF EXISTS update_context_memory_updated_at ON context_memory;
DROP TRIGGER IF EXISTS update_canvas_artifacts_updated_at ON canvas_artifacts;
DROP TRIGGER IF EXISTS update_user_patterns_updated_at ON user_patterns;

-- Drop function if no other tables use it
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS context_memory_refs;
DROP TABLE IF EXISTS user_patterns;
DROP TABLE IF EXISTS canvas_artifacts;
DROP TABLE IF EXISTS context_memory;