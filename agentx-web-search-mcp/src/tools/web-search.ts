import { z } from 'zod';
import { SearchProviderManager, SearchOptions } from '../providers/index.js';

// Input schema for web search tool
export const WebSearchInputSchema = z.object({
  query: z.string().min(1).max(500).describe('The search query'),
  maxResults: z.number().int().min(1).max(20).default(10).describe('Maximum number of results to return'),
  searchEngine: z.enum(['auto', 'bing', 'brave', 'duckduckgo']).default('auto').describe('Preferred search engine (auto uses fallback priority)'),
  includeContent: z.boolean().default(false).describe('Whether to fetch and include full page content'),
  language: z.string().optional().describe('Language code (e.g., "en", "es", "fr")'),
  region: z.string().optional().describe('Region code (e.g., "US", "GB", "DE")'),
  timeRange: z.enum(['day', 'week', 'month', 'year']).optional().describe('Time range for search results'),
  safeSearch: z.enum(['strict', 'moderate', 'off']).default('moderate').describe('Safe search filter'),
});

export type WebSearchInput = z.infer<typeof WebSearchInputSchema>;

// Output schema
export const WebSearchOutputSchema = z.object({
  results: z.array(z.object({
    title: z.string(),
    url: z.string(),
    snippet: z.string(),
    source: z.string(),
    favicon: z.string().optional(),
    timestamp: z.string().optional(),
    content: z.object({
      markdown: z.string(),
      plainText: z.string(),
      wordCount: z.number(),
      title: z.string().optional(),
      description: z.string().optional(),
    }).optional(),
  })),
  metadata: z.object({
    totalResults: z.number(),
    provider: z.string(),
    query: z.string(),
    searchTime: z.number(),
    errors: z.array(z.object({
      provider: z.string(),
      error: z.string(),
    })),
  }),
});

export type WebSearchOutput = z.infer<typeof WebSearchOutputSchema>;

export async function webSearchTool(
  input: WebSearchInput,
  searchManager: SearchProviderManager,
  contentProcessor?: any
): Promise<WebSearchOutput> {
  const startTime = Date.now();

  // Prepare search options
  const searchOptions: SearchOptions = {
    maxResults: input.maxResults,
    language: input.language,
    region: input.region,
    timeRange: input.timeRange,
    safeSearch: input.safeSearch,
  };

  let results: any[];
  let provider: string;
  let errors: Array<{ provider: string; error: string }>;

  // Execute search based on engine preference
  if (input.searchEngine === 'auto') {
    const searchResult = await searchManager.searchWithFallback(input.query, searchOptions);
    results = searchResult.results;
    provider = searchResult.provider;
    errors = searchResult.errors;
  } else {
    // Use specific provider
    const specificProvider = searchManager.getProviderByName(input.searchEngine);
    if (!specificProvider) {
      throw new Error(`Search provider "${input.searchEngine}" is not available`);
    }
    
    try {
      results = await specificProvider.search(input.query, searchOptions);
      provider = input.searchEngine;
      errors = [];
    } catch (error) {
      results = [];
      provider = 'none';
      errors = [{
        provider: input.searchEngine,
        error: error instanceof Error ? error.message : String(error),
      }];
    }
  }

  // Optionally fetch full content for each result
  const processedResults = [];
  
  for (const result of results) {
    let processedResult: any = {
      title: result.title,
      url: result.url,
      snippet: result.snippet,
      source: result.source,
      favicon: result.favicon,
      timestamp: result.timestamp,
    };

    if (input.includeContent && contentProcessor) {
      try {
        // Fetch the page content
        const response = await fetch(result.url, {
          headers: {
            'User-Agent': 'Mozilla/5.0 (compatible; AgentX-WebSearch/1.0)',
          },
          signal: AbortSignal.timeout(15000), // 15 second timeout
        });

        if (response.ok) {
          const html = await response.text();
          const processedContent = await contentProcessor.processHtml(html, result.url);
          
          processedResult.content = {
            markdown: processedContent.markdown,
            plainText: processedContent.plainText,
            wordCount: processedContent.wordCount,
            title: processedContent.title,
            description: processedContent.description,
          };
        }
      } catch (error) {
        console.warn(`Failed to fetch content for ${result.url}:`, error);
        // Continue without content
      }
    }

    processedResults.push(processedResult);
  }

  const searchTime = Date.now() - startTime;

  return {
    results: processedResults,
    metadata: {
      totalResults: processedResults.length,
      provider,
      query: input.query,
      searchTime,
      errors,
    },
  };
}