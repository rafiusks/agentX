# Feature Flags Documentation

AgentX uses feature flags to control the availability of features. This allows for gradual rollout, A/B testing, and disabling features that aren't ready for production.

## Configuration

Feature flags are controlled via environment variables. Copy `.env.example` to `.env` and set the desired values:

```bash
cp .env.example .env
```

## Available Feature Flags

### VITE_FEATURE_CONTEXT_MEMORY
- **Default**: `false`
- **Description**: Enables the Context Memory System for persistent knowledge across conversations
- **Requirements**: 
  - Backend implementation of context memory endpoints
  - User authentication
  - PostgreSQL with proper migrations
- **Status**: Backend implementation pending

### VITE_FEATURE_CANVAS_MODE
- **Default**: `true`
- **Description**: Enables the Canvas Mode for side-by-side document/code editing
- **Features**:
  - Code editor with syntax highlighting
  - Document editor with markdown support
  - Data editor for JSON
  - Version history
- **Status**: Fully implemented and enabled

### VITE_FEATURE_SMART_ACTIONS
- **Default**: `false`
- **Description**: Enables Smart Response Actions
- **Features**:
  - Format-aware clipboard operations
  - Content transformations
  - Quick actions on AI responses
- **Status**: UI implemented, backend integration pending

### VITE_FEATURE_AI_SUGGESTIONS
- **Default**: `false`
- **Description**: Enables Proactive AI Suggestions based on conversation context
- **Status**: Not yet implemented

### VITE_FEATURE_MODEL_ROUTING
- **Default**: `false`
- **Description**: Enables Smart Model Routing to automatically select the best model for each task
- **Status**: Not yet implemented

### VITE_REQUIRE_AUTH
- **Default**: `false`
- **Description**: Requires authentication for all features
- **Status**: Authentication system implemented, optional by default

## Usage in Code

### Checking Feature Flags

```typescript
import { FEATURES, isFeatureEnabled } from '@/config/features';

// Direct check
if (FEATURES.CANVAS_MODE) {
  // Canvas mode is enabled
}

// Using helper function
if (isFeatureEnabled('CONTEXT_MEMORY')) {
  // Context memory is enabled
}
```

### Conditional Rendering

```tsx
import { FEATURES } from '@/config/features';

export const MyComponent = () => {
  return (
    <>
      {FEATURES.CANVAS_MODE && (
        <CanvasComponent />
      )}
      
      {FEATURES.CONTEXT_MEMORY && (
        <MemoryIndicator />
      )}
    </>
  );
};
```

## Development Workflow

1. **New Feature Development**: Start with the flag set to `false`
2. **Testing**: Enable the flag in your local `.env` file
3. **Staging**: Enable for select users or environments
4. **Production**: Enable globally when ready
5. **Cleanup**: Remove the flag check once the feature is stable

## Troubleshooting

### Feature Not Showing
1. Check if the feature flag is enabled in your `.env` file
2. Restart the development server after changing environment variables
3. Clear browser cache if using cached builds
4. Check browser console for any errors

### 403 Forbidden Errors
- Context Memory features require authentication
- Some features may require specific backend permissions
- Check if the backend endpoints are implemented

## Future Improvements

- Dynamic feature flag management (without restart)
- User-specific feature flags
- A/B testing framework
- Feature flag dashboard in admin UI