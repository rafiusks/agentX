import axios from 'axios';
import * as cheerio from 'cheerio';
import type { Config } from '../config/index.js';
import { SearchProvider, SearchResult, SearchOptions, SearchError, NetworkError, RateLimitError } from './base.js';

export class DuckDuckGoProvider implements SearchProvider {
  name = 'DuckDuckGo';
  priority: number;
  enabled: boolean;
  private config: Config;
  private lastRequestTime = 0;
  private requestDelay = 1000; // 1 second between requests

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

      const searchUrl = this.buildSearchUrl(query, options);
      const response = await axios.get(searchUrl, {
        timeout: this.config.searchProviders.duckduckgo.timeout,
        headers: {
          'User-Agent': this.config.browser.userAgent,
          'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
          'Accept-Language': 'en-US,en;q=0.9',
          'Accept-Encoding': 'gzip, deflate, br',
          'Connection': 'keep-alive',
          'Upgrade-Insecure-Requests': '1',
        },
        validateStatus: (status) => {
          // Accept 200, 202, and 302 status codes
          return status === 200 || status === 202 || status === 302;
        },
      });

      if (response.status === 429) {
        throw new RateLimitError(this.name);
      }

      // Handle 202 Accepted (async response) by retrying
      if (response.status === 202) {
        // Wait a bit and retry
        await new Promise(resolve => setTimeout(resolve, 1000));
        const retryResponse = await axios.get(searchUrl, {
          timeout: this.config.searchProviders.duckduckgo.timeout,
          headers: {
            'User-Agent': this.config.browser.userAgent,
            'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
            'Accept-Language': 'en-US,en;q=0.9',
          },
          });
        
        if (retryResponse.status === 200) {
          return this.parseResults(String(retryResponse.data), options.maxResults);
        }
      }

      if (response.status !== 200 && response.status !== 202) {
        throw new SearchError(`HTTP ${response.status}: ${response.statusText}`, this.name, response.status);
      }

      return this.parseResults(String(response.data), options.maxResults);

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
      const response = await axios.head('https://duckduckgo.com', {
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

  private buildSearchUrl(query: string, options: SearchOptions): string {
    // Use the standard HTML endpoint which works better
    const params = new URLSearchParams({
      q: query,
    });

    if (options.language) {
      params.set('kl', this.mapLanguage(options.language));
    }

    if (options.region) {
      params.set('kl', this.mapRegion(options.region));
    }

    if (options.safeSearch) {
      params.set('safe', this.mapSafeSearch(options.safeSearch));
    }

    if (options.timeRange) {
      params.set('df', this.mapTimeRange(options.timeRange));
    }

    // Use the regular HTML endpoint
    return `https://html.duckduckgo.com/html/?${params.toString()}`;
  }

  private parseResults(html: string, maxResults = 10): SearchResult[] {
    const $ = cheerio.load(html);
    const results: SearchResult[] = [];

    // Main selector for DuckDuckGo HTML results
    $('.result').each((index, element) => {
      if (index >= maxResults) return false;

      const $result = $(element);
      
      // Find the title and URL
      const $titleLink = $result.find('.result__title a, .result__a');
      const title = $titleLink.text().trim();
      const url = $titleLink.attr('href');
      
      // Find the snippet
      const $snippet = $result.find('.result__snippet');
      const snippet = $snippet.text().trim();

      if (title && url) {
        // DuckDuckGo uses redirect URLs, extract the real URL
        const realUrl = this.extractRealUrl(url);
        
        results.push({
          title,
          url: realUrl,
          snippet: snippet || 'No description available',
          source: this.name,
        });
      }
    });

    // Alternative: try links class results
    if (results.length === 0) {
      $('#links .result, .results .result').each((index, element) => {
        if (index >= maxResults) return false;

        const $result = $(element);
        const $titleLink = $result.find('a.result__a');
        
        const title = $titleLink.text().trim();
        const url = $titleLink.attr('href');
        const snippet = $result.find('.result__snippet').text().trim();

        if (title && url) {
          results.push({
            title,
            url: this.extractRealUrl(url),
            snippet: snippet || 'No description available',
            source: this.name,
          });
        }
      });
    }

    return results;
  }

  private extractRealUrl(url: string): string {
    try {
      // Handle DuckDuckGo redirect URLs
      if (url.includes('duckduckgo.com/l/?')) {
        const urlObj = new URL(url);
        const realUrl = urlObj.searchParams.get('uddg');
        if (realUrl) {
          return decodeURIComponent(realUrl);
        }
      }
      
      // If it's already a proper URL, return as is
      if (url.startsWith('http')) {
        return url;
      }
      
      // If it's a relative URL, make it absolute
      if (url.startsWith('/')) {
        return `https://duckduckgo.com${url}`;
      }
      
      return url;
    } catch {
      return url;
    }
  }

  private mapLanguage(language: string): string {
    const languageMap: { [key: string]: string } = {
      'en': 'en-us',
      'es': 'es-es',
      'fr': 'fr-fr',
      'de': 'de-de',
      'it': 'it-it',
      'pt': 'pt-br',
      'ja': 'jp-jp',
      'ko': 'kr-kr',
      'zh': 'cn-zh',
    };
    
    return languageMap[language] || 'en-us';
  }

  private mapRegion(region: string): string {
    // DuckDuckGo uses the same format as language
    return this.mapLanguage(region);
  }

  private mapSafeSearch(safeSearch: string): string {
    const safeSearchMap: { [key: string]: string } = {
      'strict': 'strict',
      'moderate': 'moderate',
      'off': '-1',
    };
    
    return safeSearchMap[safeSearch] || 'moderate';
  }

  private mapTimeRange(timeRange: string): string {
    const timeRangeMap: { [key: string]: string } = {
      'day': 'd',
      'week': 'w',
      'month': 'm',
      'year': 'y',
    };
    
    return timeRangeMap[timeRange] || '';
  }
}