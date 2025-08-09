/**
 * Retry utility with exponential backoff for API calls
 */

export interface RetryOptions {
  maxAttempts?: number;
  initialDelay?: number;
  maxDelay?: number;
  backoffFactor?: number;
  shouldRetry?: (error: any, attempt: number) => boolean;
  onRetry?: (error: any, attempt: number, nextDelay: number) => void;
}

const DEFAULT_OPTIONS: Required<RetryOptions> = {
  maxAttempts: 3,
  initialDelay: 1000,
  maxDelay: 30000,
  backoffFactor: 2,
  shouldRetry: (error) => {
    // Retry on rate limits, server errors, and network errors
    if (error.status === 429) return true; // Rate limit
    if (error.status >= 500) return true; // Server errors
    if (error.status === 0) return true; // Network error
    if (error.code === 'ECONNREFUSED') return true;
    if (error.code === 'ETIMEDOUT') return true;
    if (error.message?.includes('fetch')) return true;
    return false;
  },
  onRetry: () => {}
};

export async function withRetry<T>(
  fn: () => Promise<T>,
  options: RetryOptions = {}
): Promise<T> {
  const opts = { ...DEFAULT_OPTIONS, ...options };
  let lastError: any;
  
  for (let attempt = 1; attempt <= opts.maxAttempts; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error;
      
      // Check if we should retry
      if (attempt === opts.maxAttempts || !opts.shouldRetry(error, attempt)) {
        throw error;
      }
      
      // Calculate delay with exponential backoff
      const delay = Math.min(
        opts.initialDelay * Math.pow(opts.backoffFactor, attempt - 1),
        opts.maxDelay
      );
      
      // Add jitter to prevent thundering herd
      const jitteredDelay = delay + Math.random() * 1000;
      
      // Notify about retry
      opts.onRetry(error, attempt, jitteredDelay);
      
      // Wait before retrying
      await new Promise(resolve => setTimeout(resolve, jitteredDelay));
    }
  }
  
  throw lastError;
}

/**
 * Specialized retry for rate limits with smart backoff
 */
export async function withRateLimitRetry<T>(
  fn: () => Promise<T>,
  onRetry?: (attempt: number, delay: number) => void
): Promise<T> {
  return withRetry(fn, {
    maxAttempts: 5,
    initialDelay: 2000,
    maxDelay: 60000,
    shouldRetry: (error) => {
      // Only retry rate limits and temporary errors
      return error.status === 429 || error.status === 503;
    },
    onRetry: (error, attempt, delay) => {
      // Check for Retry-After header
      if (error.headers?.['retry-after']) {
        const retryAfter = parseInt(error.headers['retry-after']);
        if (!isNaN(retryAfter)) {
          delay = retryAfter * 1000;
        }
      }
      
      console.log(`[Retry] Rate limited, retrying in ${Math.round(delay / 1000)}s (attempt ${attempt})`);
      onRetry?.(attempt, delay);
    }
  });
}

/**
 * Provider fallback chain
 */
export async function withProviderFallback<T>(
  providers: Array<{ name: string; fn: () => Promise<T> }>,
  onFallback?: (from: string, to: string, error: any) => void
): Promise<T> {
  let lastError: any;
  
  for (let i = 0; i < providers.length; i++) {
    const provider = providers[i];
    const nextProvider = providers[i + 1];
    
    try {
      // Try with retry for each provider
      return await withRetry(provider.fn, {
        maxAttempts: 2,
        initialDelay: 1000,
        onRetry: (error, attempt) => {
          console.log(`[${provider.name}] Retry attempt ${attempt} due to:`, error.message);
        }
      });
    } catch (error) {
      lastError = error;
      
      if (nextProvider) {
        console.log(`[Fallback] ${provider.name} failed, trying ${nextProvider.name}`);
        onFallback?.(provider.name, nextProvider.name, error);
      }
    }
  }
  
  throw new Error(`All providers failed. Last error: ${lastError?.message || 'Unknown error'}`);
}