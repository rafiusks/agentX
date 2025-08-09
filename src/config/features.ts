/**
 * Feature flags for enabling/disabling features
 * These can be controlled via environment variables or settings
 */

export const FEATURES = {
  // Context Memory System - Stores persistent knowledge across conversations
  CONTEXT_MEMORY: import.meta.env.VITE_FEATURE_CONTEXT_MEMORY === 'true' || false,
  
  // Canvas Mode - Side-by-side document/code editing
  CANVAS_MODE: import.meta.env.VITE_FEATURE_CANVAS_MODE === 'true' || true,
  
  // Smart Response Actions - Format-aware clipboard and transformations
  SMART_ACTIONS: import.meta.env.VITE_FEATURE_SMART_ACTIONS === 'true' || false,
  
  // Proactive AI Suggestions
  AI_SUGGESTIONS: import.meta.env.VITE_FEATURE_AI_SUGGESTIONS === 'true' || false,
  
  // Smart Model Routing
  MODEL_ROUTING: import.meta.env.VITE_FEATURE_MODEL_ROUTING === 'true' || false,
  
  // Authentication Required
  REQUIRE_AUTH: import.meta.env.VITE_REQUIRE_AUTH === 'true' || false,
} as const;

// Helper to check if a feature is enabled
export const isFeatureEnabled = (feature: keyof typeof FEATURES): boolean => {
  return FEATURES[feature];
};