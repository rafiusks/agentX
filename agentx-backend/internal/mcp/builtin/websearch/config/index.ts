import { z } from 'zod';

// Configuration schema
export const ConfigSchema = z.object({
  // Search providers - DuckDuckGo only (no browser needed)
  searchProviders: z.object({
    duckduckgo: z.object({
      enabled: z.boolean().default(true),
      priority: z.number().default(1),
      timeout: z.number().default(15000),
    }).default({}),
  }).default({}),

  // Browser automation settings
  browser: z.object({
    headless: z.boolean().default(true),
    userAgent: z.string().default(
      'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36'
    ),
    viewport: z.object({
      width: z.number().default(1280),
      height: z.number().default(720),
    }).default({}),
    navigationTimeout: z.number().default(30000),
  }).default({}),

  // Content processing
  content: z.object({
    maxLength: z.number().default(50000),
    maxResults: z.number().default(10),
    includeImages: z.boolean().default(false),
    preserveFormatting: z.boolean().default(true),
    removeAds: z.boolean().default(true),
    removeNavigation: z.boolean().default(true),
  }).default({}),

  // Caching and performance
  cache: z.object({
    enabled: z.boolean().default(true),
    ttl: z.number().default(300), // 5 minutes
    maxEntries: z.number().default(1000),
  }).default({}),

  // Rate limiting
  rateLimit: z.object({
    requestsPerMinute: z.number().default(30),
    requestsPerHour: z.number().default(500),
  }).default({}),

  // Logging
  logging: z.object({
    level: z.enum(['debug', 'info', 'warn', 'error']).default('info'),
    logSearchQueries: z.boolean().default(false), // Privacy consideration
  }).default({}),
});

export type Config = z.infer<typeof ConfigSchema>;

// Default configuration
export const defaultConfig: Config = ConfigSchema.parse({});

// Load configuration from environment variables
export function loadConfig(): Config {
  const envConfig = {
    searchProviders: {
      duckduckgo: {
        enabled: process.env.DUCKDUCKGO_ENABLED !== 'false',
        timeout: process.env.DUCKDUCKGO_TIMEOUT ? parseInt(process.env.DUCKDUCKGO_TIMEOUT) : undefined,
      },
    },
    browser: {
      headless: process.env.BROWSER_HEADLESS !== 'false',
      userAgent: process.env.USER_AGENT,
      navigationTimeout: process.env.NAVIGATION_TIMEOUT ? parseInt(process.env.NAVIGATION_TIMEOUT) : undefined,
    },
    content: {
      maxLength: process.env.MAX_CONTENT_LENGTH ? parseInt(process.env.MAX_CONTENT_LENGTH) : undefined,
      maxResults: process.env.MAX_SEARCH_RESULTS ? parseInt(process.env.MAX_SEARCH_RESULTS) : undefined,
    },
    logging: {
      level: (process.env.LOG_LEVEL as any) || 'info',
      logSearchQueries: process.env.LOG_SEARCH_QUERIES === 'true',
    },
  };

  return ConfigSchema.parse(envConfig);
}