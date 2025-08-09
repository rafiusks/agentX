# LLM System Migration Plan

## Overview
This document outlines the migration from the dual LLM system to a unified Gateway-based architecture.

## Current State (Dual System)
- **UnifiedChatService**: Uses providers.Registry, handles chat endpoints
- **LLMService**: Uses llm.Gateway, handles title generation
- **Problem**: Duplicate code, inconsistent routing, maintenance burden

## Target State (Unified Gateway)
- **Single System**: All LLM operations through llm.Gateway
- **User-Scoped**: Per-user provider isolation for security
- **Advanced Features**: Circuit breakers, metrics, sophisticated routing

## Migration Phases

### Phase 1: Adapter Layer ✅ COMPLETE
- Created `UnifiedChatAdapter` that wraps llm.Gateway
- Maintains API compatibility with existing handlers
- All chat operations now route through Gateway

### Phase 2: Handler Migration (Next)
1. Update handlers to use LLMService directly
2. Remove dependency on UnifiedChatService interface
3. Add user context to all LLM operations

### Phase 3: Cleanup (Final)
1. Remove old UnifiedChatService
2. Remove providers.Registry (keep only for Gateway initialization)
3. Remove RequestRouter
4. Consolidate all routing logic in llm.Router

## Benefits of Migration
1. **Single Source of Truth**: One system for all LLM operations
2. **Better Security**: User-scoped provider management
3. **Advanced Features**: Circuit breakers, metrics, fallbacks
4. **Maintainability**: Less code duplication, clearer architecture
5. **Extensibility**: Easier to add new providers and features

## Testing Strategy
1. Verify chat endpoints work with adapter
2. Verify title generation continues to work
3. Test provider initialization and routing
4. Performance testing to ensure no regression

## Rollback Plan
If issues arise, the adapter can be replaced with the original UnifiedChatService by reverting services.go changes.

## Code Locations
- Adapter: `/internal/services/unified_chat_adapter.go`
- Gateway: `/internal/llm/gateway.go`
- Old Service: `/internal/services/unified_chat.go` (to be removed)
- Handlers: `/internal/api/handlers/unified_chat.go` (to be updated)

## Timeline
- Phase 1: Immediate (Complete)
- Phase 2: 1-2 days
- Phase 3: 3-5 days

## Success Metrics
- All endpoints functional
- No performance regression
- Reduced code complexity
- Improved test coverage