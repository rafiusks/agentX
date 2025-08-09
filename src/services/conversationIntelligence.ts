
// Simplified context that's automatically extracted from conversations
export interface ConversationContext {
  entities: {
    people: string[];
    projects: string[];
    technologies: string[];
    files: string[];
  };
  preferences: Map<string, string>;
  recentTopics: string[];
  sessionId: string;
}

// Lightweight context extractor using simple heuristics
class ConversationIntelligenceService {
  private activeContext: ConversationContext | null = null;
  private contextCache = new Map<string, ConversationContext>();
  
  // Simple patterns for entity extraction (no ML needed)
  private patterns = {
    people: /\b[A-Z][a-z]+ [A-Z][a-z]+\b/g, // Names like "John Doe"
    projects: /\b(project|app|application|system|platform|service)[\s:]+([a-zA-Z0-9-_]+)/gi,
    technologies: /\b(React|Vue|Angular|Node|Python|Go|Rust|TypeScript|JavaScript|Docker|Kubernetes|AWS|Azure|GCP)\b/gi,
    files: /\b[\w-]+\.(tsx?|jsx?|py|go|rs|md|json|yaml|yml|css|scss)\b/g,
    preferences: /\b(prefer|like|want|need|should|always|never)\s+(.+?)(?:\.|,|;|$)/gi
  };
  
  // Extract context from a message automatically
  extractFromMessage(content: string): Partial<ConversationContext['entities']> {
    const entities: Partial<ConversationContext['entities']> = {
      people: [],
      projects: [],
      technologies: [],
      files: []
    };
    
    // Extract people names
    const peopleMatches = content.match(this.patterns.people) || [];
    entities.people = [...new Set(peopleMatches)];
    
    // Extract project references
    const projectMatches = content.matchAll(this.patterns.projects);
    for (const match of projectMatches) {
      if (match[2]) entities.projects?.push(match[2]);
    }
    entities.projects = [...new Set(entities.projects)];
    
    // Extract technologies
    const techMatches = content.match(this.patterns.technologies) || [];
    entities.technologies = [...new Set(techMatches)];
    
    // Extract file references
    const fileMatches = content.match(this.patterns.files) || [];
    entities.files = [...new Set(fileMatches)];
    
    return entities;
  }
  
  // Update context based on conversation flow
  async updateContext(sessionId: string, messages: Array<{ role: string; content: string }>) {
    if (!this.activeContext || this.activeContext.sessionId !== sessionId) {
      this.activeContext = {
        entities: { people: [], projects: [], technologies: [], files: [] },
        preferences: new Map(),
        recentTopics: [],
        sessionId
      };
    }
    
    // Process last few messages for context
    const recentMessages = messages.slice(-10); // Only look at recent context
    
    for (const message of recentMessages) {
      const extracted = this.extractFromMessage(message.content);
      
      // Merge extracted entities
      if (extracted.people) {
        this.activeContext.entities.people = [
          ...new Set([...this.activeContext.entities.people, ...extracted.people])
        ];
      }
      if (extracted.projects) {
        this.activeContext.entities.projects = [
          ...new Set([...this.activeContext.entities.projects, ...extracted.projects])
        ];
      }
      if (extracted.technologies) {
        this.activeContext.entities.technologies = [
          ...new Set([...this.activeContext.entities.technologies, ...extracted.technologies])
        ];
      }
      if (extracted.files) {
        this.activeContext.entities.files = [
          ...new Set([...this.activeContext.entities.files, ...extracted.files])
        ];
      }
    }
    
    // Cache the context
    this.contextCache.set(sessionId, this.activeContext);
  }
  
  // Get relevant context for current conversation
  getActiveContext(sessionId: string): ConversationContext | null {
    return this.contextCache.get(sessionId) || null;
  }
  
  // Check if context is being used (for subtle indicator)
  isContextActive(sessionId: string): boolean {
    const context = this.contextCache.get(sessionId);
    if (!context) return false;
    
    // Context is active if we have extracted entities
    return (
      context.entities.people.length > 0 ||
      context.entities.projects.length > 0 ||
      context.entities.technologies.length > 0 ||
      context.entities.files.length > 0
    );
  }
  
  // Get context summary for tooltip
  getContextSummary(sessionId: string): string {
    const context = this.contextCache.get(sessionId);
    if (!context) return 'No context detected';
    
    const parts: string[] = [];
    
    if (context.entities.people.length > 0) {
      parts.push(`People: ${context.entities.people.slice(0, 3).join(', ')}`);
    }
    if (context.entities.projects.length > 0) {
      parts.push(`Projects: ${context.entities.projects.slice(0, 3).join(', ')}`);
    }
    if (context.entities.technologies.length > 0) {
      parts.push(`Tech: ${context.entities.technologies.slice(0, 3).join(', ')}`);
    }
    if (context.entities.files.length > 0) {
      parts.push(`Files: ${context.entities.files.slice(0, 3).join(', ')}`);
    }
    
    return parts.length > 0 ? parts.join(' • ') : 'Monitoring conversation...';
  }
  
  // Clear context for a session
  clearContext(sessionId: string) {
    this.contextCache.delete(sessionId);
    if (this.activeContext?.sessionId === sessionId) {
      this.activeContext = null;
    }
  }
  
  // Get smart suggestions based on context - be very selective
  getSuggestions(sessionId: string): string[] {
    const context = this.contextCache.get(sessionId);
    if (!context) return [];
    
    const suggestions: string[] = [];
    
    // Only suggest for very specific, actionable scenarios
    
    // Multiple error-related files mentioned - likely debugging
    const errorFiles = context.entities.files.filter(f => 
      f.includes('error') || f.includes('exception') || f.includes('.log')
    );
    if (errorFiles.length >= 2) {
      suggestions.push('I can help analyze those error logs for patterns');
    }
    
    // Package files mentioned with multiple technologies - likely dependency issues
    const hasPackageFile = context.entities.files.some(f => 
      f.includes('package.json') || f.includes('requirements.txt') || f.includes('go.mod')
    );
    if (hasPackageFile && context.entities.technologies.length >= 3) {
      suggestions.push('Would you like me to check for dependency conflicts?');
    }
    
    // Multiple people mentioned - likely collaboration scenario
    if (context.entities.people.length >= 3) {
      suggestions.push('Should I help draft a team update or documentation?');
    }
    
    // Return only the most relevant suggestion, not multiple
    return suggestions.slice(0, 1);
  }
}

export const conversationIntelligence = new ConversationIntelligenceService();