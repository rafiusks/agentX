# AgentX Web Search MCP Server

A robust Model Context Protocol (MCP) server for web search capabilities, designed for AgentX and other MCP-compatible clients. Features multiple search providers with intelligent fallback, content extraction, and summarization.

## Features

### 🔍 Multi-Provider Search
- **Bing Search**: Browser-based search with Playwright for high-quality results
- **Brave Search**: API and web scraping support with privacy focus  
- **DuckDuckGo**: Fallback HTTP-based search for reliability
- **Smart Fallback**: Automatically tries providers in priority order

### 🛠 Three Powerful Tools

#### 1. `web_search`
General web search with configurable options:
- Multiple search engines with automatic fallback
- Optional full content extraction from results
- Customizable result count, language, region
- Time range filtering and safe search controls

#### 2. `fetch_page`
Extract content from specific URLs:
- Multiple output formats (Markdown, plain text, HTML)
- Smart content cleaning (removes ads, navigation)
- Link and image extraction
- Security protections against local/private URLs

#### 3. `search_and_summarize`
Research tool that searches and summarizes:
- Fetches content from top search results
- Generates intelligent summaries with relevance scoring
- Configurable summary length (brief/detailed/comprehensive)
- Includes source attribution and links

### ⚡ Advanced Features
- **Content Processing**: HTML → Markdown conversion with smart cleaning
- **Rate Limiting**: Respect search engine limits and robots.txt
- **Error Handling**: Comprehensive error recovery and fallback
- **Caching**: Optional result caching for performance
- **Security**: Blocks access to private/local network addresses

## Installation

### Prerequisites
- Node.js 18+
- TypeScript (for development)

### Install Dependencies
```bash
npm install
```

### Build
```bash
npm run build
```

## Configuration

Configure via environment variables:

### Search Providers
```bash
# Bing (default: enabled)
BING_ENABLED=true
BING_TIMEOUT=30000

# Brave (default: enabled)
BRAVE_ENABLED=true
BRAVE_API_KEY=your_api_key_here  # Optional: enables API mode
BRAVE_TIMEOUT=30000

# DuckDuckGo (default: enabled)
DUCKDUCKGO_ENABLED=true
DUCKDUCKGO_TIMEOUT=15000
```

### Browser Settings
```bash
# Browser automation
BROWSER_HEADLESS=true
USER_AGENT="Mozilla/5.0 (compatible; AgentX-WebSearch/1.0)"
NAVIGATION_TIMEOUT=30000
```

### Content Processing
```bash
# Content limits
MAX_CONTENT_LENGTH=50000
MAX_SEARCH_RESULTS=10

# Logging
LOG_LEVEL=info
LOG_SEARCH_QUERIES=false  # Privacy: don't log queries by default
```

## Usage with AgentX

### 1. Start the Server
```bash
npm start
```

### 2. Add to AgentX MCP Configuration

Add this server to your AgentX MCP servers configuration:

```json
{
  "name": "Web Search",
  "description": "Robust web search with multiple providers and content extraction",
  "command": "node",
  "args": ["/path/to/agentx-web-search-mcp/dist/index.js"],
  "env": {
    "LOG_LEVEL": "info"
  },
  "enabled": true
}
```

### 3. Available Tools in AgentX

Once connected, AgentX will have access to:

- **`web_search`**: "Search the web with multiple engines"
- **`fetch_page`**: "Extract content from any webpage" 
- **`search_and_summarize`**: "Research topics with automatic summarization"

## Tool Examples

### Web Search
```json
{
  "query": "artificial intelligence latest developments 2024",
  "maxResults": 10,
  "searchEngine": "auto",
  "includeContent": false,
  "timeRange": "month"
}
```

### Fetch Page
```json
{
  "url": "https://example.com/article",
  "format": "markdown",
  "maxLength": 10000,
  "includeLinks": true
}
```

### Search and Summarize
```json
{
  "query": "climate change renewable energy solutions",
  "maxPages": 5,
  "summaryLength": "detailed",
  "includeLinks": true
}
```

## Development

### Run in Development Mode
```bash
npm run dev
```

### Testing
```bash
npm test
```

### Linting
```bash
npm run lint
```

## Architecture

```
src/
├── index.ts              # Main MCP server
├── config/               # Configuration management
├── providers/            # Search provider implementations
│   ├── base.ts          # Provider interface
│   ├── bing.ts          # Bing search via Playwright
│   ├── brave.ts         # Brave search (API + scraping)
│   ├── duckduckgo.ts    # DuckDuckGo HTTP search
│   └── index.ts         # Provider manager
├── content/              # Content processing
│   └── processor.ts     # HTML → Markdown conversion
└── tools/               # MCP tool implementations
    ├── web-search.ts    # General web search tool
    ├── fetch-page.ts    # Page content extraction
    └── search-and-summarize.ts  # Research summarization
```

## Error Handling

The server includes comprehensive error handling:

- **Network Errors**: Automatic retry with exponential backoff
- **Rate Limiting**: Respects provider limits with intelligent delays  
- **Content Errors**: Graceful fallback when pages can't be processed
- **Provider Failures**: Automatic fallback to next available provider

## Security

- **URL Validation**: Prevents access to local/private network addresses
- **Content Limits**: Configurable limits prevent abuse
- **Rate Limiting**: Built-in delays respect service terms
- **No Persistence**: Search queries are not stored permanently

## Performance

- **Concurrent Processing**: Parallel content fetching where safe
- **Smart Caching**: Optional caching of search results
- **Content Optimization**: Efficient HTML processing and cleaning
- **Provider Prioritization**: Fastest providers tried first

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Submit a pull request

## License

MIT License - see LICENSE file for details.