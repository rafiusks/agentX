/**
 * Shared API type definitions
 */

// Error types
export interface ApiErrorResponse {
  error?: string;
  message?: string;
  details?: Record<string, unknown>;
}

// Provider types
export interface ProviderConfig {
  api_key?: string;
  base_url?: string;
  organization?: string;
  model?: string;
  models?: string[];
  default_model?: string;
  [key: string]: string | number | boolean | string[] | undefined;
}

// Connection types
export interface ConnectionConfig {
  provider_id: string;
  name: string;
  config: ProviderConfig;
}

export interface ConnectionUpdate {
  name?: string;
  config?: ProviderConfig;
  enabled?: boolean;
  is_default?: boolean;
}

// Settings types
export interface AppSettings {
  theme?: 'light' | 'dark' | 'auto';
  default_provider?: string;
  default_model?: string;
  streaming_enabled?: boolean;
  auto_save?: boolean;
  [key: string]: string | number | boolean | undefined;
}

// Message types
export interface ChatMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
  metadata?: Record<string, unknown>;
}

// Stream chunk types
export interface StreamChunk {
  id?: string;
  object?: string;
  created?: number;
  model?: string;
  choices?: Array<{
    index: number;
    delta?: {
      role?: string;
      content?: string;
    };
    finish_reason?: string | null;
  }>;
  usage?: {
    prompt_tokens?: number;
    completion_tokens?: number;
    total_tokens?: number;
  };
}

// Memory types
export interface ContextMemoryData {
  content: string;
  metadata?: Record<string, unknown>;
  importance?: number;
  namespace?: string;
  expires_at?: string;
}

// Canvas types
export interface CanvasContent {
  type: 'code' | 'document' | 'data';
  content: string;
  language?: string;
  title?: string;
}

// Auth types
export interface LoginCredentials {
  email: string;
  password: string;
}

export interface SignupData extends LoginCredentials {
  confirmPassword: string;
}

export interface AuthTokens {
  access_token: string;
  refresh_token?: string;
}

export interface UserProfile {
  id: string;
  email: string;
  name?: string;
  created_at: string;
  updated_at: string;
}

// API key types
export interface ApiKeyData {
  name: string;
  provider: string;
  key: string;
  expiresAt?: string;
}

export interface ApiKeyUpdate {
  name?: string;
  key?: string;
  expiresAt?: string;
}

// Function call types
export interface FunctionCall {
  name: string;
  arguments: Record<string, unknown>;
  id?: string;
}

// Generic response types
export type ApiResponse<T> = T | ApiErrorResponse;

// Pagination types
export interface PaginationParams {
  page?: number;
  limit?: number;
  sort?: string;
  order?: 'asc' | 'desc';
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  pages: number;
}