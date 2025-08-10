-- Drop triggers
DROP TRIGGER IF EXISTS update_mcp_resources_updated_at ON mcp_resources;
DROP TRIGGER IF EXISTS update_mcp_tools_updated_at ON mcp_tools;
DROP TRIGGER IF EXISTS update_mcp_servers_updated_at ON mcp_servers;

-- Drop indexes
DROP INDEX IF EXISTS idx_mcp_resources_server_id;
DROP INDEX IF EXISTS idx_mcp_tools_enabled;
DROP INDEX IF EXISTS idx_mcp_tools_server_id;
DROP INDEX IF EXISTS idx_mcp_servers_status;
DROP INDEX IF EXISTS idx_mcp_servers_enabled;
DROP INDEX IF EXISTS idx_mcp_servers_user_id;

-- Drop tables
DROP TABLE IF EXISTS mcp_resources;
DROP TABLE IF EXISTS mcp_tools;
DROP TABLE IF EXISTS mcp_servers;