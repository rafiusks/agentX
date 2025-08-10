import axios from 'axios';
import { firefox } from 'playwright';
import * as cheerio from 'cheerio';
import type { Config } from '../config/index.js';
import { SearchProvider, SearchResult, SearchOptions, SearchError, NetworkError, RateLimitError } from './base.js';

export class BraveProvider implements SearchProvider {
  name = 'Brave';
  priority: number;
  enabled: boolean;
  private config: Config;
  private lastRequestTime = 0;
  private requestDelay = 1500; // 1.5 seconds between requests

  constructor(config: Config) {
    this.config = config;
    this.priority = config.searchProviders.brave.priority;
    this.enabled = config.searchProviders.brave.enabled;
  }

  async search(query: string, options: SearchOptions = {}): Promise<SearchResult[]> {
    if (!this.enabled) {
      throw new SearchError('Brave provider is disabled', this.name);
    }

    // Try API first if API key is available
    if (this.config.searchProviders.brave.apiKey) {
      try {
        return await this.searchWithApi(query, options);
      } catch (error) {
        // Fallback to web scraping if API fails
        console.error('Brave API search failed, falling back to web scraping:', error);
      }
    }

    // Fallback to web scraping
    return await this.searchWithScraping(query, options);
  }

  private async searchWithApi(query: string, options: SearchOptions = {}): Promise<SearchResult[]> {
    try {
      await this.enforceRateLimit();

      const params: any = {
        q: query,
        count: Math.min(options.maxResults || 10, 20), // Brave API max is 20
        offset: 0,
        safesearch: this.mapSafeSearch(options.safeSearch || 'moderate'),
        search_lang: options.language || 'en',
        country: options.region || 'US',
      };

      if (options.timeRange) {
        params.freshness = this.mapTimeRangeForApi(options.timeRange);
      }

      const response = await axios.get('https://api.search.brave.com/res/v1/web/search', {
        headers: {
          'Accept': 'application/json',
          'Accept-Encoding': 'gzip',
          'X-Subscription-Token': this.config.searchProviders.brave.apiKey!,
        },
        params,
        timeout: this.config.searchProviders.brave.timeout,
      });

      if (response.status === 429) {
        throw new RateLimitError(this.name);
      }

      if (response.status !== 200) {
        throw new SearchError(`HTTP ${response.status}: ${response.statusText}`, this.name, response.status);
      }

      return this.parseApiResults(response.data);

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
        if ((error as any).response?.status === 401) {
          throw new SearchError('Invalid API key', this.name, 401);
        }
        throw new NetworkError(this.name, error as Error);
      }

      throw new SearchError(`Unexpected API error: ${String(error)}`, this.name, undefined, error instanceof Error ? error : new Error(String(error)));
    }
  }

  private async searchWithScraping(query: string, options: SearchOptions = {}): Promise<SearchResult[]> {
    const browser = await firefox.launch({
      headless: this.config.browser.headless,
      args: ['--no-sandbox', '--disable-setuid-sandbox'],
    });

    try {
      await this.enforceRateLimit();

      const page = await browser.newPage();
      await page.setExtraHTTPHeaders({
        'User-Agent': this.config.browser.userAgent,
      });
      await page.setViewportSize(this.config.browser.viewport);

      const searchUrl = this.buildSearchUrl(query, options);
      
      const response = await page.goto(searchUrl, {
        timeout: this.config.browser.navigationTimeout,
        waitUntil: 'networkidle',
      });

      if (!response) {
        throw new SearchError('Failed to load search page', this.name);
      }

      if (response.status() === 429) {
        throw new RateLimitError(this.name);
      }

      if (!response.ok()) {
        throw new SearchError(`HTTP ${response.status()}: ${response.statusText()}`, this.name, response.status());
      }

      // Wait for results to load
      try {
        await page.waitForSelector('.snippet', { timeout: 10000 });
      } catch {
        // Continue if selector not found
      }

      const html = await page.content();
      return this.parseWebResults(html, options.maxResults);

    } catch (error) {
      if (error instanceof SearchError) {
        throw error;
      }

      if ((error as any).name === 'TimeoutError') {
        throw new NetworkError(this.name, new Error('Request timeout'));
      }

      throw new SearchError(`Web scraping error: ${String(error)}`, this.name, undefined, error instanceof Error ? error : new Error(String(error)));

    } finally {
      await browser.close();
    }
  }

  async isAvailable(): Promise<boolean> {
    try {
      // Test API if available
      if (this.config.searchProviders.brave.apiKey) {
        const response = await axios.head('https://api.search.brave.com/res/v1/web/search', {
          headers: {
            'X-Subscription-Token': this.config.searchProviders.brave.apiKey,
          },
          timeout: 5000,
        });
        return response.status === 200;
      }

      // Test web interface
      const response = await axios.head('https://search.brave.com', {
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
    const params = new URLSearchParams({
      q: query,
      source: 'web',
    });

    if (options.language) {
      params.set('country', options.language.toUpperCase());
    }

    if (options.region) {
      params.set('country', options.region.toUpperCase());
    }

    if (options.safeSearch) {
      params.set('safesearch', this.mapSafeSearch(options.safeSearch));
    }

    return `https://search.brave.com/search?${params.toString()}`;
  }

  private parseApiResults(data: any): SearchResult[] {
    const results: SearchResult[] = [];

    if (!data.web || !data.web.results) {
      return results;
    }

    for (const result of data.web.results) {
      if (result.title && result.url && result.description) {
        results.push({
          title: result.title,
          url: result.url,
          snippet: result.description,
          source: this.name,
          favicon: result.profile?.img,
        });
      }
    }

    return results;
  }

  private parseWebResults(html: string, maxResults = 10): SearchResult[] {
    const $ = cheerio.load(html);
    const results: SearchResult[] = [];

    // Primary selector for Brave search results
    $('.snippet').each((index, element) => {
      if (index >= maxResults) return false;

      const $result = $(element);
      const $titleLink = $result.find('.snippet-title a, .title a');
      const $snippet = $result.find('.snippet-description, .description');
      
      const title = $titleLink.text().trim();
      const url = $titleLink.attr('href');
      const snippet = $snippet.text().trim();

      if (title && url && snippet) {
        results.push({
          title,
          url,
          snippet,
          source: this.name,
        });
      }
    });

    // Fallback selector
    if (results.length === 0) {
      $('[data-type="web"] .result').each((index, element) => {
        if (index >= maxResults) return false;

        const $result = $(element);
        const $titleLink = $result.find('.result-header a');
        const $snippet = $result.find('.snippet-content, .result-description');
        
        const title = $titleLink.text().trim();
        const url = $titleLink.attr('href');
        const snippet = $snippet.text().trim();

        if (title && url && snippet) {
          results.push({
            title,
            url,
            snippet,
            source: this.name,
          });
        }
      });
    }

    return results;
  }

  private mapSafeSearch(safeSearch: string): string {
    const safeSearchMap: { [key: string]: string } = {
      'strict': 'strict',
      'moderate': 'moderate', 
      'off': 'off',
    };
    
    return safeSearchMap[safeSearch] || 'moderate';
  }

  private mapTimeRangeForApi(timeRange: string): string {
    const timeRangeMap: { [key: string]: string } = {
      'day': 'pd',
      'week': 'pw',
      'month': 'pm',
      'year': 'py',
    };
    
    return timeRangeMap[timeRange] || '';
  }
}