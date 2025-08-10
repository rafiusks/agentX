#!/usr/bin/env node

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
  ErrorCode,
  McpError,
} from '@modelcontextprotocol/sdk/types.js';

import { loadConfig } from './config/index.js';
import { SearchProviderManager } from './providers/index.js';
import { ContentProcessor } from './content/processor.js';

// Import tools
import { 
  webSearchTool, 
  WebSearchInputSchema, 
  WebSearchInput 
} from './tools/web-search.js';
import { 
  fetchPageTool, 
  FetchPageInputSchema, 
  FetchPageInput 
} from './tools/fetch-page.js';
import { 
  searchAndSummarizeTool, 
  SearchAndSummarizeInputSchema, 
  SearchAndSummarizeInput 
} from './tools/search-and-summarize.js';

class WebSearchMCPServer {
  private server: Server;
  private config: ReturnType<typeof loadConfig>;
  private searchManager: SearchProviderManager;
  private contentProcessor: ContentProcessor;

  constructor() {
    // Load configuration
    this.config = loadConfig();
    console.error('[WebSearchMCP] Configuration loaded');

    // Initialize components
    this.searchManager = new SearchProviderManager(this.config);
    this.contentProcessor = new ContentProcessor(this.config);

    // Create server
    this.server = new Server(
      {
        name: 'agentx-web-search-mcp',
        version: '1.0.0',
      },
      {
        capabilities: {
          tools: {},
        },
      }
    );

    this.setupToolHandlers();
    console.error('[WebSearchMCP] Server initialized');
  }

  private setupToolHandlers(): void {
    // List available tools
    this.server.setRequestHandler(ListToolsRequestSchema, async () => {
      return {
        tools: [
          {
            name: 'web_search',
            description: 'Search the web using multiple search engines with fallback support. Returns search results with optional full content extraction.',
            inputSchema: WebSearchInputSchema,
          },
          {
            name: 'fetch_page',
            description: 'Fetch and extract content from a specific web page URL. Supports multiple output formats and content filtering.',
            inputSchema: FetchPageInputSchema,
          },
          {
            name: 'search_and_summarize',
            description: 'Perform a web search and automatically fetch, process, and summarize content from the top results. Perfect for research tasks.',
            inputSchema: SearchAndSummarizeInputSchema,
          },
        ],
      };
    });

    // Handle tool calls
    this.server.setRequestHandler(CallToolRequestSchema, async (request) => {
      const { name, arguments: args } = request.params;

      try {
        switch (name) {
          case 'web_search':
            return await this.handleWebSearch(args as WebSearchInput);
          
          case 'fetch_page':
            return await this.handleFetchPage(args as FetchPageInput);
          
          case 'search_and_summarize':
            return await this.handleSearchAndSummarize(args as SearchAndSummarizeInput);
          
          default:
            throw new McpError(
              ErrorCode.MethodNotFound,
              `Unknown tool: ${name}`
            );
        }
      } catch (error) {
        console.error(`[WebSearchMCP] Tool ${name} failed:`, error);
        
        if (error instanceof McpError) {
          throw error;
        }
        
        throw new McpError(
          ErrorCode.InternalError,
          `Tool execution failed: ${error instanceof Error ? error.message : 'Unknown error'}`
        );
      }
    });
  }

  private async handleWebSearch(args: WebSearchInput) {
    console.error(`[WebSearchMCP] Web search: "${args.query}"`);
    
    // Validate input
    const validatedInput = WebSearchInputSchema.parse(args);
    
    // Execute search
    const result = await webSearchTool(
      validatedInput, 
      this.searchManager,
      validatedInput.includeContent ? this.contentProcessor : undefined
    );

    console.error(`[WebSearchMCP] Search completed: ${result.metadata.totalResults} results from ${result.metadata.provider}`);

    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(result, null, 2),
        },
      ],
    };
  }

  private async handleFetchPage(args: FetchPageInput) {
    console.error(`[WebSearchMCP] Fetch page: ${args.url}`);
    
    // Validate input
    const validatedInput = FetchPageInputSchema.parse(args);
    
    // Fetch and process page
    const result = await fetchPageTool(validatedInput, this.contentProcessor);

    console.error(`[WebSearchMCP] Page fetched: ${result.metadata.wordCount} words, ${result.metadata.contentLength} chars`);

    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(result, null, 2),
        },
      ],
    };
  }

  private async handleSearchAndSummarize(args: SearchAndSummarizeInput) {
    console.error(`[WebSearchMCP] Search and summarize: "${args.query}"`);
    
    // Validate input
    const validatedInput = SearchAndSummarizeInputSchema.parse(args);
    
    // Execute search and summarization
    const result = await searchAndSummarizeTool(
      validatedInput,
      this.searchManager,
      this.contentProcessor
    );

    console.error(`[WebSearchMCP] Summarization completed: processed ${result.metadata.processedPages} pages, ${result.metadata.summaryWordCount} word summary`);

    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(result, null, 2),
        },
      ],
    };
  }

  async run(): Promise<void> {
    const transport = new StdioServerTransport();
    await this.server.connect(transport);
    console.error('[WebSearchMCP] Server running on stdio');
  }

  async cleanup(): Promise<void> {
    console.error('[WebSearchMCP] Cleaning up...');
    await this.searchManager.cleanup();
  }
}

// Main execution
async function main() {
  const server = new WebSearchMCPServer();
  
  // Handle graceful shutdown
  const cleanup = async () => {
    console.error('[WebSearchMCP] Received shutdown signal');
    await server.cleanup();
    process.exit(0);
  };

  process.on('SIGINT', cleanup);
  process.on('SIGTERM', cleanup);
  process.on('SIGQUIT', cleanup);

  // Handle uncaught errors
  process.on('uncaughtException', (error) => {
    console.error('[WebSearchMCP] Uncaught exception:', error);
    cleanup();
  });

  process.on('unhandledRejection', (reason, promise) => {
    console.error('[WebSearchMCP] Unhandled rejection at:', promise, 'reason:', reason);
    cleanup();
  });

  try {
    await server.run();
  } catch (error) {
    console.error('[WebSearchMCP] Failed to start server:', error);
    process.exit(1);
  }
}

// Only run if this file is executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch((error) => {
    console.error('[WebSearchMCP] Fatal error:', error);
    process.exit(1);
  });
}