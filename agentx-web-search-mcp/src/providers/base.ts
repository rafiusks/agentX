export interface SearchResult {
  title: string;
  url: string;
  snippet: string;
  favicon?: string;
  timestamp?: string;
  source: string; // The search provider that returned this result
}

export interface SearchOptions {
  maxResults?: number;
  language?: string;
  region?: string;
  timeRange?: 'day' | 'week' | 'month' | 'year';
  safeSearch?: 'strict' | 'moderate' | 'off';
}

export interface SearchProvider {
  name: string;
  priority: number;
  enabled: boolean;
  
  search(query: string, options?: SearchOptions): Promise<SearchResult[]>;
  isAvailable(): Promise<boolean>;
  getHealthStatus(): Promise<{ healthy: boolean; latency?: number; error?: string }>;
}

export class SearchError extends Error {
  constructor(
    message: string,
    public provider: string,
    public statusCode?: number,
    public originalError?: Error
  ) {
    super(message);
    this.name = 'SearchError';
  }
}

export class RateLimitError extends SearchError {
  constructor(provider: string, retryAfter?: number) {
    super(`Rate limit exceeded for ${provider}`, provider, 429);
    this.name = 'RateLimitError';
  }
}

export class NetworkError extends SearchError {
  constructor(provider: string, originalError: Error) {
    super(`Network error in ${provider}: ${originalError.message}`, provider, undefined, originalError);
    this.name = 'NetworkError';
  }
}