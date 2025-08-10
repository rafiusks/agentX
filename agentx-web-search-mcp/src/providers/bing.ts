import { chromium, Browser, Page } from 'playwright';
import * as cheerio from 'cheerio';
import type { Config } from '../config/index.js';
import { SearchProvider, SearchResult, SearchOptions, SearchError, NetworkError, RateLimitError } from './base.js';

export class BingProvider implements SearchProvider {
  name = 'Bing';
  priority: number;
  enabled: boolean;
  private config: Config;
  private browser: Browser | null = null;
  private lastRequestTime = 0;
  private requestDelay = 2000; // 2 seconds between requests

  constructor(config: Config) {
    this.config = config;
    this.priority = config.searchProviders.bing.priority;
    this.enabled = config.searchProviders.bing.enabled;
  }

  async search(query: string, options: SearchOptions = {}): Promise<SearchResult[]> {
    if (!this.enabled) {
      throw new SearchError('Bing provider is disabled', this.name);
    }

    let page: Page | null = null;

    try {
      // Rate limiting
      await this.enforceRateLimit();

      // Initialize browser if needed
      if (!this.browser) {
        await this.initializeBrowser();
      }

      if (!this.browser) {
        throw new SearchError('Failed to initialize browser', this.name);
      }

      page = await this.browser.newPage();

      // Set user agent and viewport
      await page.setExtraHTTPHeaders({
        'User-Agent': this.config.browser.userAgent,
      });
      await page.setViewportSize(this.config.browser.viewport);

      // Navigate to Bing search
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
        await page.waitForSelector('#b_results', { timeout: 10000 });
      } catch {
        // If we can't find results container, continue anyway
      }

      // Get page content
      const html = await page.content();
      
      return this.parseResults(html, options.maxResults);

    } catch (error) {
      if (error instanceof SearchError) {
        throw error;
      }

      if ((error as any).name === 'TimeoutError') {
        throw new NetworkError(this.name, new Error('Request timeout'));
      }

      throw new SearchError(`Unexpected error: ${String(error)}`, this.name, undefined, error instanceof Error ? error : new Error(String(error)));
      
    } finally {
      if (page) {
        try {
          await page.close();
        } catch (e) {
          // Ignore close errors
        }
      }
    }
  }

  async isAvailable(): Promise<boolean> {
    try {
      if (!this.browser) {
        await this.initializeBrowser();
      }

      if (!this.browser) {
        return false;
      }

      const page = await this.browser.newPage();
      const response = await page.goto('https://www.bing.com', { timeout: 5000 });
      await page.close();
      
      return response?.ok() ?? false;
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
        error: available ? undefined : 'Service unreachable or browser initialization failed',
      };
    } catch (error) {
      return {
        healthy: false,
        latency: Date.now() - startTime,
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  async cleanup(): Promise<void> {
    if (this.browser) {
      try {
        await this.browser.close();
      } catch (e) {
        // Ignore close errors
      }
      this.browser = null;
    }
  }

  private async initializeBrowser(): Promise<void> {
    try {
      this.browser = await chromium.launch({
        headless: this.config.browser.headless,
        args: [
          '--no-sandbox',
          '--disable-setuid-sandbox',
          '--disable-dev-shm-usage',
          '--disable-accelerated-2d-canvas',
          '--no-first-run',
          '--no-zygote',
          '--disable-gpu',
        ],
      });
    } catch (error) {
      throw new SearchError('Failed to launch browser', this.name, undefined, error as Error);
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
      form: 'QBLH',
    });

    if (options.language) {
      params.set('setlang', options.language);
    }

    if (options.region) {
      params.set('cc', options.region);
    }

    if (options.safeSearch) {
      params.set('safesearch', this.mapSafeSearch(options.safeSearch));
    }

    if (options.timeRange) {
      params.set('filters', `ex1:"ez5_${this.mapTimeRange(options.timeRange)}"`);
    }

    return `https://www.bing.com/search?${params.toString()}`;
  }

  private parseResults(html: string, maxResults = 10): SearchResult[] {
    const $ = cheerio.load(html);
    const results: SearchResult[] = [];

    // Primary selector for Bing results
    $('#b_results .b_algo').each((index, element) => {
      if (index >= maxResults) return false;

      const $result = $(element);
      const $titleLink = $result.find('h2 a');
      const $snippet = $result.find('.b_caption p, .b_caption .b_dList, .b_caption .b_snippet');
      
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

    // Fallback selector for different Bing layouts
    if (results.length === 0) {
      $('.b_algo').each((index, element) => {
        if (index >= maxResults) return false;

        const $result = $(element);
        const $titleLink = $result.find('a[href]').first();
        const $snippet = $result.find('p, .b_snippet');
        
        const title = $titleLink.text().trim();
        const url = $titleLink.attr('href');
        const snippet = $snippet.text().trim();

        if (title && url && snippet && !url.includes('bing.com/search')) {
          results.push({
            title,
            url,
            snippet,
            source: this.name,
          });
        }
      });
    }

    // Another fallback for mobile/alternative layouts
    if (results.length === 0) {
      $('.b_searchResult').each((index, element) => {
        if (index >= maxResults) return false;

        const $result = $(element);
        const $titleLink = $result.find('.b_title a, h2 a');
        const $snippet = $result.find('.b_snippet, .b_caption');
        
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

  private mapTimeRange(timeRange: string): string {
    const timeRangeMap: { [key: string]: string } = {
      'day': 'qdr_d',
      'week': 'qdr_w', 
      'month': 'qdr_m',
      'year': 'qdr_y',
    };
    
    return timeRangeMap[timeRange] || '';
  }
}