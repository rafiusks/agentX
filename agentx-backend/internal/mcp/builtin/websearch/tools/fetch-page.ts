import { z } from 'zod';
import axios from 'axios';

// Input schema for fetch page tool
export const FetchPageInputSchema = z.object({
  url: z.string().url().describe('The URL to fetch and extract content from'),
  format: z.enum(['markdown', 'text', 'html']).default('markdown').describe('Output format'),
  maxLength: z.number().int().min(100).max(100000).default(50000).describe('Maximum content length'),
  includeLinks: z.boolean().default(true).describe('Whether to include extracted links'),
  includeImages: z.boolean().default(false).describe('Whether to include image information'),
  timeout: z.number().int().min(5000).max(60000).default(30000).describe('Request timeout in milliseconds'),
});

export type FetchPageInput = z.infer<typeof FetchPageInputSchema>;

// Output schema
export const FetchPageOutputSchema = z.object({
  content: z.string(),
  metadata: z.object({
    url: z.string(),
    title: z.string().optional(),
    description: z.string().optional(),
    wordCount: z.number(),
    format: z.string(),
    fetchTime: z.number(),
    contentLength: z.number(),
    links: z.array(z.object({
      text: z.string(),
      url: z.string(),
    })).optional(),
    images: z.array(z.object({
      alt: z.string(),
      src: z.string(),
    })).optional(),
  }),
  error: z.string().optional(),
});

export type FetchPageOutput = z.infer<typeof FetchPageOutputSchema>;

export async function fetchPageTool(
  input: FetchPageInput,
  contentProcessor: any
): Promise<FetchPageOutput> {
  const startTime = Date.now();

  try {
    // Validate and clean URL
    const url = new URL(input.url);
    
    // Security check - block local/private IPs
    if (isPrivateOrLocalUrl(url)) {
      throw new Error('Access to local or private URLs is not allowed');
    }

    // Fetch the page
    const response = await axios.get(url.href, {
      timeout: input.timeout,
      headers: {
        'User-Agent': 'Mozilla/5.0 (compatible; AgentX-WebSearch/1.0; +https://agentx.ai)',
        'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
        'Accept-Language': 'en-US,en;q=0.9',
        'Accept-Encoding': 'gzip, deflate, br',
        'Connection': 'keep-alive',
        'Upgrade-Insecure-Requests': '1',
      },
      responseType: 'text',
      // maxContentLength: 10 * 1024 * 1024, // 10MB limit - not supported in axios
    });

    if (!response.data) {
      throw new Error('No content received from URL');
    }

    // Process the HTML content
    const processedContent = await contentProcessor.processHtml(response.data, url.href);

    // Format content based on requested format
    let outputContent: string;
    switch (input.format) {
      case 'markdown':
        outputContent = processedContent.markdown;
        break;
      case 'text':
        outputContent = processedContent.plainText;
        break;
      case 'html':
        outputContent = String(response.data);
        break;
      default:
        outputContent = processedContent.markdown;
    }

    // Truncate if necessary
    if (outputContent.length > input.maxLength) {
      const truncateAt = input.maxLength - 50; // Leave room for truncation message
      if (input.format === 'markdown') {
        outputContent = truncateContent(outputContent, truncateAt) + '\n\n[Content truncated...]';
      } else {
        outputContent = outputContent.substring(0, truncateAt) + '\n\n[Content truncated...]';
      }
    }

    const fetchTime = Date.now() - startTime;

    return {
      content: outputContent,
      metadata: {
        url: url.href,
        title: processedContent.title,
        description: processedContent.description,
        wordCount: processedContent.wordCount,
        format: input.format,
        fetchTime,
        contentLength: outputContent.length,
        links: input.includeLinks ? processedContent.links : undefined,
        images: input.includeImages ? processedContent.images : undefined,
      },
    };

  } catch (error) {
    const fetchTime = Date.now() - startTime;
    let errorMessage = 'Unknown error occurred';
    
    if ((axios as any).isAxiosError && (axios as any).isAxiosError(error)) {
      if ((error as any).code === 'ECONNABORTED') {
        errorMessage = 'Request timeout - the server took too long to respond';
      } else if ((error as any).response) {
        errorMessage = `HTTP ${(error as any).response.status}: ${(error as any).response.statusText}`;
      } else if ((error as any).request) {
        errorMessage = 'Network error - unable to reach the server';
      } else {
        errorMessage = (error as Error).message;
      }
    } else if (error instanceof Error) {
      errorMessage = error.message;
    }

    return {
      content: '',
      metadata: {
        url: input.url,
        wordCount: 0,
        format: input.format,
        fetchTime,
        contentLength: 0,
      },
      error: errorMessage,
    };
  }
}

// Helper function to check for private/local URLs
function isPrivateOrLocalUrl(url: URL): boolean {
  const hostname = url.hostname.toLowerCase();
  
  // Check for localhost variants
  if (hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '::1') {
    return true;
  }
  
  // Check for private IP ranges
  const ipRegex = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/;
  const ipMatch = hostname.match(ipRegex);
  
  if (ipMatch) {
    const [, a, b, c, d] = ipMatch.map(Number);
    
    // Private IP ranges
    if (
      (a === 10) ||
      (a === 172 && b >= 16 && b <= 31) ||
      (a === 192 && b === 168) ||
      (a === 169 && b === 254) || // Link-local
      (a === 0) || // Current network
      (a === 127) // Loopback
    ) {
      return true;
    }
  }
  
  // Check for .local domains and other local TLDs
  if (hostname.endsWith('.local') || hostname.endsWith('.localhost')) {
    return true;
  }
  
  return false;
}

// Helper function to truncate content at word boundaries
function truncateContent(content: string, maxLength: number): string {
  if (content.length <= maxLength) {
    return content;
  }

  // Try to truncate at a natural break point
  const truncated = content.substring(0, maxLength);
  
  // Look for paragraph break
  const lastParagraph = truncated.lastIndexOf('\n\n');
  if (lastParagraph > maxLength * 0.8) {
    return content.substring(0, lastParagraph);
  }
  
  // Look for sentence break
  const lastSentence = truncated.lastIndexOf('. ');
  if (lastSentence > maxLength * 0.8) {
    return content.substring(0, lastSentence + 1);
  }
  
  // Look for word break
  const lastSpace = truncated.lastIndexOf(' ');
  if (lastSpace > maxLength * 0.9) {
    return content.substring(0, lastSpace);
  }
  
  return truncated;
}