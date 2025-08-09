// Context management configuration
export interface ContextConfig {
  maxMessages: number;
  maxCharacters: number;
  strategy: 'sliding_window' | 'smart' | 'full';
  includeSystemMessages: boolean;
  excludeErrors: boolean;
  useImportanceScoring?: boolean;
  importanceThreshold?: number; // Min score to include old messages
}

// Default context configuration
export const DEFAULT_CONTEXT_CONFIG: ContextConfig = {
  maxMessages: 20,        // Last 20 messages (roughly 10 exchanges)
  maxCharacters: 12000,   // ~3000 tokens for most models
  strategy: 'sliding_window',
  includeSystemMessages: true,
  excludeErrors: true,
  useImportanceScoring: false,
  importanceThreshold: 0.6,
};

// Provider-specific configurations
export const PROVIDER_CONTEXT_LIMITS: Record<string, Partial<ContextConfig>> = {
  'openai': {
    maxMessages: 50,
    maxCharacters: 30000,  // GPT-4 can handle more
  },
  'openai-compatible': {
    maxMessages: 20,
    maxCharacters: 12000,  // Conservative for local models
  },
  'anthropic': {
    maxMessages: 100,
    maxCharacters: 80000,  // Claude has 200k context
  },
  'local': {
    maxMessages: 10,
    maxCharacters: 6000,   // Most local models have smaller contexts
  },
  'ollama': {
    maxMessages: 10,
    maxCharacters: 6000,
  },
};

// Get context config based on provider
export function getContextConfig(provider?: string): ContextConfig {
  const providerConfig = provider ? PROVIDER_CONTEXT_LIMITS[provider] : {};
  return {
    ...DEFAULT_CONTEXT_CONFIG,
    ...providerConfig,
  };
}