import type { Config } from '../config/index.js';
import { SearchProvider, SearchResult, SearchOptions, SearchError } from './base.js';
import { DuckDuckGoProvider } from './duckduckgo.js';

export class SearchProviderManager {
  private providers: SearchProvider[] = [];
  private config: Config;

  constructor(config: Config) {
    this.config = config;
    this.initializeProviders();
  }

  private initializeProviders(): void {
    // Initialize DuckDuckGo provider
    const duckduckgo = new DuckDuckGoProvider(this.config);

    // Add enabled providers and sort by priority
    const allProviders = [duckduckgo];
    this.providers = allProviders
      .filter(provider => provider.enabled)
      .sort((a, b) => a.priority - b.priority);

    if (this.providers.length === 0) {
      console.warn('No search providers are enabled!');
    }
  }

  async search(query: string, options: SearchOptions = {}): Promise<SearchResult[]> {
    const errors: { provider: string; error: Error }[] = [];
    const maxResults = options.maxResults || this.config.content.maxResults;

    for (const provider of this.providers) {
      try {
        console.log(`[SearchManager] Trying ${provider.name} for query: "${query}"`);
        
        const results = await provider.search(query, { ...options, maxResults });
        
        if (results.length > 0) {
          console.log(`[SearchManager] ${provider.name} returned ${results.length} results`);
          
          // Deduplicate results by URL
          const deduplicatedResults = this.deduplicateResults(results);
          
          if (deduplicatedResults.length > 0) {
            return deduplicatedResults.slice(0, maxResults);
          }
        }

        console.log(`[SearchManager] ${provider.name} returned no usable results, trying next provider`);
        
      } catch (error) {
        console.error(`[SearchManager] ${provider.name} failed:`, error);
        errors.push({ 
          provider: provider.name, 
          error: error instanceof Error ? error : new Error(String(error))
        });
        
        // Continue to next provider unless this is a critical error
        continue;
      }
    }

    // If all providers failed, throw an aggregate error
    if (errors.length === this.providers.length) {
      const errorMessages = errors.map(e => `${e.provider}: ${e.error.message}`).join('; ');
      throw new SearchError(`All search providers failed: ${errorMessages}`, 'SearchProviderManager');
    }

    // If some providers failed but we got here, it means no results were found
    throw new SearchError(`No search results found for query: "${query}"`, 'SearchProviderManager');
  }

  async searchWithFallback(query: string, options: SearchOptions = {}): Promise<{
    results: SearchResult[];
    provider: string;
    errors: Array<{ provider: string; error: string }>;
  }> {
    const errors: Array<{ provider: string; error: string }> = [];
    const maxResults = options.maxResults || this.config.content.maxResults;

    for (const provider of this.providers) {
      try {
        const results = await provider.search(query, { ...options, maxResults });
        
        if (results.length > 0) {
          return {
            results: this.deduplicateResults(results).slice(0, maxResults),
            provider: provider.name,
            errors,
          };
        }
        
      } catch (error) {
        errors.push({
          provider: provider.name,
          error: error instanceof Error ? error.message : String(error),
        });
      }
    }

    return {
      results: [],
      provider: 'none',
      errors,
    };
  }

  async getProviderHealthStatus(): Promise<Array<{
    name: string;
    enabled: boolean;
    healthy: boolean;
    latency?: number;
    error?: string;
  }>> {
    const statuses = [];

    for (const provider of this.providers) {
      try {
        const health = await provider.getHealthStatus();
        statuses.push({
          name: provider.name,
          enabled: provider.enabled,
          ...health,
        });
      } catch (error) {
        statuses.push({
          name: provider.name,
          enabled: provider.enabled,
          healthy: false,
          error: error instanceof Error ? error.message : 'Unknown error',
        });
      }
    }

    return statuses;
  }

  getEnabledProviders(): string[] {
    return this.providers.map(p => p.name);
  }

  getProviderByName(name: string): SearchProvider | null {
    return this.providers.find(p => p.name.toLowerCase() === name.toLowerCase()) || null;
  }

  private deduplicateResults(results: SearchResult[]): SearchResult[] {
    const seen = new Set<string>();
    const deduplicated: SearchResult[] = [];

    for (const result of results) {
      // Normalize URL for comparison
      const normalizedUrl = this.normalizeUrl(result.url);
      
      if (!seen.has(normalizedUrl)) {
        seen.add(normalizedUrl);
        deduplicated.push(result);
      }
    }

    return deduplicated;
  }

  private normalizeUrl(url: string): string {
    try {
      const urlObj = new URL(url);
      // Remove common tracking parameters
      const trackingParams = ['utm_source', 'utm_medium', 'utm_campaign', 'utm_content', 'utm_term', 'fbclid', 'gclid'];
      
      trackingParams.forEach(param => {
        urlObj.searchParams.delete(param);
      });

      // Remove trailing slash and fragment
      let normalizedUrl = urlObj.origin + urlObj.pathname + urlObj.search;
      if (normalizedUrl.endsWith('/')) {
        normalizedUrl = normalizedUrl.slice(0, -1);
      }

      return normalizedUrl;
    } catch {
      return url;
    }
  }

  async cleanup(): Promise<void> {
    for (const provider of this.providers) {
      if ('cleanup' in provider && typeof provider.cleanup === 'function') {
        try {
          await provider.cleanup();
        } catch (error) {
          console.error(`Error cleaning up ${provider.name}:`, error);
        }
      }
    }
  }
}

// Export provider types and classes
export { SearchProvider, SearchResult, SearchOptions, SearchError } from './base.js';
export { DuckDuckGoProvider } from './duckduckgo.js';