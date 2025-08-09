# Migration Notes

## Migration 007 - Context Memory System

### Issue Fixed (2025-08-08)
The migration was failing with a "Dirty database version 7" error due to the pgvector extension not being available in the PostgreSQL container.

### Resolution
1. **Problem**: The migration tried to use `vector(1536)` type for embeddings, which requires the pgvector extension.
2. **Solution**: Modified the migration to use `TEXT` type temporarily for the embedding column until pgvector can be properly installed.

### Steps to Fix Dirty Migration
If you encounter a dirty migration error:

```bash
# 1. Check migration status
docker exec agentx-backend-postgres-1 psql -U agentx -d agentx -c "SELECT * FROM schema_migrations;"

# 2. Mark migration as not dirty
docker exec agentx-backend-postgres-1 psql -U agentx -d agentx -c "UPDATE schema_migrations SET dirty = false WHERE version = <version>;"

# 3. If needed, roll back to previous version
docker exec agentx-backend-postgres-1 psql -U agentx -d agentx -c "UPDATE schema_migrations SET version = <previous_version>, dirty = false;"

# 4. Clean up any partially created tables
docker exec agentx-backend-postgres-1 psql -U agentx -d agentx -c "DROP TABLE IF EXISTS <table_name> CASCADE;"
```

### Future Enhancement
To enable vector embeddings for semantic search:
1. Use a PostgreSQL image with pgvector pre-installed
2. Update docker-compose.yml to use `ankane/pgvector` image
3. Revert the embedding column back to `vector(1536)` type

### Tables Created
- `context_memory` - Stores persistent knowledge across conversations
- `canvas_artifacts` - Stores iterative work artifacts (code, documents, etc.)
- `user_patterns` - Tracks user behavior patterns for proactive assistance
- `context_memory_refs` - Links memories to sessions and messages