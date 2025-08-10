import TurndownService from 'turndown';
import * as cheerio from 'cheerio';
import type { Config } from '../config/index.js';

export interface ProcessedContent {
  markdown: string;
  plainText: string;
  title?: string;
  description?: string;
  url: string;
  wordCount: number;
  links: Array<{ text: string; url: string }>;
  images: Array<{ alt: string; src: string }>;
}

export class ContentProcessor {
  private turndownService: TurndownService;
  private config: Config;

  constructor(config: Config) {
    this.config = config;
    this.turndownService = new TurndownService({
      headingStyle: 'atx',
      codeBlockStyle: 'fenced',
      emDelimiter: '*',
      strongDelimiter: '**',
      linkStyle: 'inlined',
    });

    // Configure Turndown to preserve important content
    this.turndownService.addRule('removeScript', {
      filter: ['script', 'style', 'noscript'],
      replacement: () => '',
    });

    this.turndownService.addRule('removeAds', {
      filter: (node: any) => {
        if (node.nodeType !== 1) return false;
        const element = node as Element;
        const classes = element.className || '';
        const id = element.id || '';
        
        // Common ad and navigation selectors
        const adPatterns = [
          'ad', 'advertisement', 'sponsor', 'promo',
          'sidebar', 'footer', 'header', 'nav',
          'menu', 'breadcrumb', 'pagination',
          'social', 'share', 'comment', 'related'
        ];
        
        return adPatterns.some(pattern => 
          classes.toLowerCase().includes(pattern) || 
          id.toLowerCase().includes(pattern)
        );
      },
      replacement: () => this.config.content.removeAds ? '' : '',
    });
  }

  async processHtml(html: string, url: string): Promise<ProcessedContent> {
    const $ = cheerio.load(html);
    
    // Extract metadata
    const title = $('title').text().trim() || 
                 $('h1').first().text().trim() || 
                 'Untitled';
    
    const description = $('meta[name="description"]').attr('content') ||
                       $('meta[property="og:description"]').attr('content') ||
                       '';

    // Remove unwanted elements if configured
    if (this.config.content.removeAds) {
      $(this.getAdSelectors().join(', ')).remove();
    }

    if (this.config.content.removeNavigation) {
      $(this.getNavigationSelectors().join(', ')).remove();
    }

    // Extract main content
    const contentSelectors = [
      'article',
      'main',
      '[role="main"]',
      '.main-content',
      '.content',
      '.post-content',
      '.article-content',
      '#content',
      '.entry-content'
    ];

    let contentElement = $('body');
    for (const selector of contentSelectors) {
      const found = $(selector);
      if (found.length > 0 && found.text().trim().length > 100) {
        contentElement = found.first() as any;
        break;
      }
    }

    // Extract links
    const links: Array<{ text: string; url: string }> = [];
    contentElement.find('a[href]').each((_, element) => {
      const $link = $(element);
      const text = $link.text().trim();
      const href = $link.attr('href');
      
      if (text && href && text.length > 0 && text.length < 200) {
        try {
          const linkUrl = new URL(href, url).href;
          links.push({ text, url: linkUrl });
        } catch (e) {
          // Invalid URL, skip
        }
      }
    });

    // Extract images
    const images: Array<{ alt: string; src: string }> = [];
    if (this.config.content.includeImages) {
      contentElement.find('img[src]').each((_, element) => {
        const $img = $(element);
        const alt = $img.attr('alt') || '';
        const src = $img.attr('src');
        
        if (src) {
          try {
            const imgUrl = new URL(src, url).href;
            images.push({ alt, src: imgUrl });
          } catch (e) {
            // Invalid URL, skip
          }
        }
      });
    }

    // Convert to markdown
    const contentHtml = contentElement.html() || '';
    let markdown = this.turndownService.turndown(contentHtml);
    
    // Clean up markdown
    markdown = this.cleanMarkdown(markdown);
    
    // Truncate if necessary
    if (markdown.length > this.config.content.maxLength) {
      markdown = this.truncateContent(markdown, this.config.content.maxLength);
    }

    const plainText = this.markdownToPlainText(markdown);
    const wordCount = this.countWords(plainText);

    return {
      markdown,
      plainText,
      title,
      description,
      url,
      wordCount,
      links: links.slice(0, 20), // Limit links
      images: images.slice(0, 10), // Limit images
    };
  }

  private getAdSelectors(): string[] {
    return [
      '.advertisement', '.ad', '.ads', '.sponsor', '.promo',
      '.social-share', '.share-buttons', '.social-media',
      '.comment', '.comments', '.comment-section',
      '.related-posts', '.related-articles', '.sidebar',
      '.popup', '.modal', '.overlay', '.banner',
      '[class*="ad-"]', '[id*="ad-"]',
      '[class*="advertisement"]', '[id*="advertisement"]',
    ];
  }

  private getNavigationSelectors(): string[] {
    return [
      'nav', 'navigation', '.nav', '.navigation',
      'header', '.header', 'footer', '.footer',
      '.menu', '.breadcrumb', '.breadcrumbs',
      '.pagination', '.pager', '.page-numbers',
      '.skip-link', '.screen-reader-text',
    ];
  }

  private cleanMarkdown(markdown: string): string {
    return markdown
      // Remove excessive whitespace
      .replace(/\n\s*\n\s*\n/g, '\n\n')
      // Remove empty links
      .replace(/\[[\s]*\]\([^)]*\)/g, '')
      // Remove excessive asterisks/underscores
      .replace(/[\*_]{3,}/g, '**')
      // Clean up list formatting
      .replace(/^\s*[\*\-\+]\s*$/gm, '')
      .trim();
  }

  private truncateContent(content: string, maxLength: number): string {
    if (content.length <= maxLength) {
      return content;
    }

    // Try to truncate at a natural break point
    const truncated = content.substring(0, maxLength);
    const lastParagraph = truncated.lastIndexOf('\n\n');
    const lastSentence = truncated.lastIndexOf('. ');
    
    let cutoff = maxLength;
    if (lastParagraph > maxLength * 0.8) {
      cutoff = lastParagraph;
    } else if (lastSentence > maxLength * 0.8) {
      cutoff = lastSentence + 1;
    }

    return content.substring(0, cutoff) + '\n\n[Content truncated...]';
  }

  private markdownToPlainText(markdown: string): string {
    return markdown
      .replace(/^#+\s*/gm, '') // Headers
      .replace(/\*\*(.+?)\*\*/g, '$1') // Bold
      .replace(/\*(.+?)\*/g, '$1') // Italic
      .replace(/\[(.+?)\]\(.+?\)/g, '$1') // Links
      .replace(/`(.+?)`/g, '$1') // Inline code
      .replace(/```[\s\S]*?```/g, '[Code Block]') // Code blocks
      .replace(/^\s*[\*\-\+]\s+/gm, '• ') // List items
      .replace(/^\s*\d+\.\s+/gm, '• ') // Numbered lists
      .replace(/\n\s*\n/g, '\n') // Collapse whitespace
      .trim();
  }

  private countWords(text: string): number {
    return text.trim().split(/\s+/).filter(word => word.length > 0).length;
  }
}