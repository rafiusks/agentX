import axios from 'axios';
import * as cheerio from 'cheerio';
import type { Config } from '../config/index.js';
import { SearchProvider, SearchResult, SearchOptions, SearchError, NetworkError, RateLimitError } from './base.js';

export class DuckDuckGoLiteProvider implements SearchProvider {
  name = 'DuckDuckGo';
  priority: number;
  enabled: boolean;
  private config: Config;
  private lastRequestTime = 0;
  private requestDelay = 500; // 0.5 second between requests

  constructor(config: Config) {
    this.config = config;
    this.priority = config.searchProviders.duckduckgo.priority;
    this.enabled = config.searchProviders.duckduckgo.enabled;
  }

  async search(query: string, options: SearchOptions = {}): Promise<SearchResult[]> {
    if (!this.enabled) {
      throw new SearchError('DuckDuckGo provider is disabled', this.name);
    }

    try {
      // Rate limiting
      await this.enforceRateLimit();

      // Use the lite version which is more reliable
      const searchUrl = `https://lite.duckduckgo.com/lite/?q=${encodeURIComponent(query)}`;
      
      const response = await axios.get(searchUrl, {
        timeout: this.config.searchProviders.duckduckgo.timeout,
        headers: {
          'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36',
          'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
          'Accept-Language': 'en-US,en;q=0.9',
        },
      });

      if (response.status === 429) {
        throw new RateLimitError(this.name);
      }

      if (response.status !== 200) {
        throw new SearchError(`HTTP ${response.status}: ${response.statusText}`, this.name, response.status);
      }

      return this.parseLiteResults(String(response.data), options.maxResults);

    } catch (error) {
      if (error instanceof SearchError) {
        throw error;
      }

      if ((axios as any).isAxiosError && (axios as any).isAxiosError(error)) {
        if ((error as any).code === 'ECONNABORTED') {
          throw new NetworkError(this.name, new Error('Request timeout'));
        }
        if ((error as any).response?.status === 429) {
          throw new RateLimitError(this.name);
        }
        throw new NetworkError(this.name, error as Error);
      }

      throw new SearchError(`Unexpected error: ${String(error)}`, this.name, undefined, error instanceof Error ? error : new Error(String(error)));
    }
  }

  async isAvailable(): Promise<boolean> {
    try {
      const response = await axios.head('https://lite.duckduckgo.com', {
        timeout: 5000,
        headers: { 'User-Agent': this.config.browser.userAgent },
      });
      return response.status === 200;
    } catch {
      return false;
    }
  }

  async getHealthStatus(): Promise<{ healthy: boolean; latency?: number; error?: string }> {
    const startTime = Date.now();
    
    try {
      const available = await this.isAvailable();
      const latency = Date.now() - startTime;
      
      return {
        healthy: available,
        latency,
        error: available ? undefined : 'Service unreachable',
      };
    } catch (error) {
      return {
        healthy: false,
        latency: Date.now() - startTime,
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  private async enforceRateLimit(): Promise<void> {
    const timeSinceLastRequest = Date.now() - this.lastRequestTime;
    if (timeSinceLastRequest < this.requestDelay) {
      const waitTime = this.requestDelay - timeSinceLastRequest;
      await new Promise(resolve => setTimeout(resolve, waitTime));
    }
    this.lastRequestTime = Date.now();
  }

  private parseLiteResults(html: string, maxResults = 10): SearchResult[] {
    const $ = cheerio.load(html);
    const results: SearchResult[] = [];

    // DuckDuckGo Lite has a simpler structure
    $('table.result-link').each((index, element) => {
      if (index >= maxResults) return false;

      const $result = $(element);
      const $link = $result.find('a.result-link');
      const $snippet = $result.find('td.result-snippet');
      
      const title = $link.text().trim();
      const url = $link.attr('href');
      const snippet = $snippet.text().trim();

      if (title && url) {
        results.push({
          title,
          url: url.startsWith('//') ? `https:${url}` : url,
          snippet: snippet || 'No description available',
          source: this.name,
        });
      }
    });

    // Alternative selector for DuckDuckGo Lite
    if (results.length === 0) {
      // Try to find results in the simple list format
      $('a.result-link').each((index, element) => {
        if (index >= maxResults) return false;

        const $link = $(element);
        const title = $link.text().trim();
        const url = $link.attr('href');
        
        // Try to find the snippet - it might be in the next element
        const $parent = $link.closest('tr, div');
        const snippet = $parent.next().find('.result-snippet, .snippet').text().trim() ||
                       $parent.find('.result-snippet, .snippet').text().trim() ||
                       'No description available';

        if (title && url) {
          results.push({
            title,
            url: url.startsWith('//') ? `https:${url}` : url,
            snippet,
            source: this.name,
          });
        }
      });
    }

    return results;
  }
}