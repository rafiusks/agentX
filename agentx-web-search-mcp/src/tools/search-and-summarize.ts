import { z } from 'zod';
import { SearchProviderManager, SearchOptions } from '../providers/index.js';

// Input schema for search and summarize tool
export const SearchAndSummarizeInputSchema = z.object({
  query: z.string().min(1).max(500).describe('The search query'),
  maxPages: z.number().int().min(1).max(10).default(5).describe('Maximum number of pages to fetch and analyze'),
  summaryLength: z.enum(['brief', 'detailed', 'comprehensive']).default('detailed').describe('Length of summary to generate'),
  includeLinks: z.boolean().default(true).describe('Whether to include relevant links in the summary'),
  language: z.string().optional().describe('Language code for search and content'),
  timeRange: z.enum(['day', 'week', 'month', 'year']).optional().describe('Time range for search results'),
});

export type SearchAndSummarizeInput = z.infer<typeof SearchAndSummarizeInputSchema>;

// Output schema
export const SearchAndSummarizeOutputSchema = z.object({
  summary: z.string(),
  sources: z.array(z.object({
    title: z.string(),
    url: z.string(),
    snippet: z.string(),
    wordCount: z.number(),
    relevanceScore: z.number().min(0).max(1),
  })),
  metadata: z.object({
    query: z.string(),
    totalSources: z.number(),
    processedPages: z.number(),
    searchTime: z.number(),
    processingTime: z.number(),
    summaryWordCount: z.number(),
    errors: z.array(z.string()),
  }),
});

export type SearchAndSummarizeOutput = z.infer<typeof SearchAndSummarizeOutputSchema>;

export async function searchAndSummarizeTool(
  input: SearchAndSummarizeInput,
  searchManager: SearchProviderManager,
  contentProcessor: any
): Promise<SearchAndSummarizeOutput> {
  const startTime = Date.now();
  const errors: string[] = [];

  try {
    // Step 1: Perform search
    const searchOptions: SearchOptions = {
      maxResults: Math.min(input.maxPages * 2, 20), // Get more results than needed for better selection
      language: input.language,
      timeRange: input.timeRange,
      safeSearch: 'moderate',
    };

    const searchResult = await searchManager.searchWithFallback(input.query, searchOptions);
    
    if (searchResult.results.length === 0) {
      throw new Error('No search results found');
    }

    // Add any search errors
    errors.push(...searchResult.errors.map(e => `${e.provider}: ${e.error}`));

    const searchTime = Date.now() - startTime;
    const processingStartTime = Date.now();

    // Step 2: Fetch and process content from top results
    const processedSources = [];
    let processedCount = 0;

    for (const result of searchResult.results.slice(0, input.maxPages)) {
      try {
        // Fetch page content
        const response = await fetch(result.url, {
          headers: {
            'User-Agent': 'Mozilla/5.0 (compatible; AgentX-WebSearch/1.0)',
            'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
          },
          signal: AbortSignal.timeout(20000), // 20 second timeout per page
        });

        if (!response.ok) {
          errors.push(`Failed to fetch ${result.url}: HTTP ${response.status}`);
          continue;
        }

        const html = await response.text();
        const processedContent = await contentProcessor.processHtml(html, result.url);

        // Calculate relevance score based on query terms
        const relevanceScore = calculateRelevanceScore(
          input.query,
          processedContent.title || result.title,
          processedContent.plainText,
          result.snippet
        );

        processedSources.push({
          title: processedContent.title || result.title,
          url: result.url,
          snippet: result.snippet,
          content: processedContent.markdown,
          plainText: processedContent.plainText,
          wordCount: processedContent.wordCount,
          relevanceScore,
          source: result.source,
        });

        processedCount++;

      } catch (error) {
        const errorMsg = `Failed to process ${result.url}: ${error instanceof Error ? error.message : 'Unknown error'}`;
        errors.push(errorMsg);
        console.warn(errorMsg);
      }
    }

    if (processedSources.length === 0) {
      throw new Error('No pages could be successfully processed');
    }

    // Sort by relevance score
    processedSources.sort((a, b) => b.relevanceScore - a.relevanceScore);

    // Step 3: Create comprehensive summary
    const summary = createSummary(
      input.query,
      processedSources,
      input.summaryLength,
      input.includeLinks
    );

    const processingTime = Date.now() - processingStartTime;
    const totalTime = Date.now() - startTime;

    return {
      summary,
      sources: processedSources.map(source => ({
        title: source.title,
        url: source.url,
        snippet: source.snippet,
        wordCount: source.wordCount,
        relevanceScore: source.relevanceScore,
      })),
      metadata: {
        query: input.query,
        totalSources: searchResult.results.length,
        processedPages: processedCount,
        searchTime,
        processingTime,
        summaryWordCount: countWords(summary),
        errors,
      },
    };

  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error occurred';
    errors.push(`Critical error: ${errorMessage}`);

    return {
      summary: `Error: Unable to generate summary. ${errorMessage}`,
      sources: [],
      metadata: {
        query: input.query,
        totalSources: 0,
        processedPages: 0,
        searchTime: Date.now() - startTime,
        processingTime: 0,
        summaryWordCount: 0,
        errors,
      },
    };
  }
}

function calculateRelevanceScore(query: string, title: string, content: string, snippet: string): number {
  const queryTerms = query.toLowerCase().split(/\s+/).filter(term => term.length > 2);
  
  if (queryTerms.length === 0) return 0.5;

  let titleScore = 0;
  let contentScore = 0;
  let snippetScore = 0;

  const titleWords = title.toLowerCase().split(/\s+/);
  const contentWords = content.toLowerCase().split(/\s+/);
  const snippetWords = snippet.toLowerCase().split(/\s+/);

  // Calculate scores based on term frequency
  for (const term of queryTerms) {
    // Title matches are worth more
    titleScore += titleWords.filter(word => word.includes(term)).length * 3;
    
    // Content matches
    contentScore += contentWords.filter(word => word.includes(term)).length;
    
    // Snippet matches are also valuable
    snippetScore += snippetWords.filter(word => word.includes(term)).length * 2;
  }

  // Normalize scores
  const maxTitleScore = queryTerms.length * titleWords.length * 3;
  const maxContentScore = queryTerms.length * Math.min(contentWords.length, 1000); // Cap content scoring
  const maxSnippetScore = queryTerms.length * snippetWords.length * 2;

  const normalizedTitleScore = maxTitleScore > 0 ? titleScore / maxTitleScore : 0;
  const normalizedContentScore = maxContentScore > 0 ? contentScore / maxContentScore : 0;
  const normalizedSnippetScore = maxSnippetScore > 0 ? snippetScore / maxSnippetScore : 0;

  // Weighted combination
  const finalScore = (
    normalizedTitleScore * 0.4 +
    normalizedContentScore * 0.4 +
    normalizedSnippetScore * 0.2
  );

  return Math.min(1, Math.max(0, finalScore));
}

function createSummary(
  query: string,
  sources: any[],
  summaryLength: 'brief' | 'detailed' | 'comprehensive',
  includeLinks: boolean
): string {
  const wordLimits = {
    brief: 200,
    detailed: 500,
    comprehensive: 1000,
  };

  const targetWords = wordLimits[summaryLength];
  
  // Start with an introduction
  let summary = `# Search Summary: ${query}\n\n`;
  
  // Add overview
  summary += `Based on ${sources.length} sources, here's what I found:\n\n`;

  // Group and synthesize information from top sources
  const topSources = sources.slice(0, summaryLength === 'brief' ? 3 : summaryLength === 'detailed' ? 5 : 8);
  
  // Extract key points from each source
  const keyPoints: string[] = [];
  
  for (const source of topSources) {
    // Extract the most relevant sentences
    const sentences = extractKeySentences(source.plainText, query, 2);
    if (sentences.length > 0) {
      const sourceInfo = includeLinks ? `[${source.title}](${source.url})` : source.title;
      keyPoints.push(`**From ${sourceInfo}:**\n${sentences.join(' ')}`);
    }
  }

  // Add key points to summary
  if (keyPoints.length > 0) {
    summary += keyPoints.join('\n\n') + '\n\n';
  }

  // Add source links if requested
  if (includeLinks && summaryLength !== 'brief') {
    summary += '## Sources\n\n';
    topSources.forEach((source, index) => {
      summary += `${index + 1}. [${source.title}](${source.url})\n`;
    });
  }

  // Ensure summary doesn't exceed target length
  const currentWordCount = countWords(summary);
  if (currentWordCount > targetWords) {
    summary = truncateSummary(summary, targetWords);
  }

  return summary.trim();
}

function extractKeySentences(text: string, query: string, maxSentences: number): string[] {
  const sentences = text.split(/[.!?]+/).map(s => s.trim()).filter(s => s.length > 20);
  const queryTerms = query.toLowerCase().split(/\s+/).filter(term => term.length > 2);
  
  // Score sentences based on query term frequency
  const scoredSentences = sentences.map(sentence => {
    const lowerSentence = sentence.toLowerCase();
    let score = 0;
    
    for (const term of queryTerms) {
      const occurrences = (lowerSentence.match(new RegExp(term, 'g')) || []).length;
      score += occurrences;
    }
    
    return { sentence, score };
  }).filter(item => item.score > 0);
  
  // Sort by score and return top sentences
  return scoredSentences
    .sort((a, b) => b.score - a.score)
    .slice(0, maxSentences)
    .map(item => item.sentence);
}

function truncateSummary(summary: string, targetWords: number): string {
  const words = summary.split(/\s+/);
  if (words.length <= targetWords) {
    return summary;
  }
  
  const truncated = words.slice(0, targetWords - 10).join(' ');
  return truncated + '\n\n[Summary truncated due to length limits]';
}

function countWords(text: string): number {
  return text.trim().split(/\s+/).filter(word => word.length > 0).length;
}