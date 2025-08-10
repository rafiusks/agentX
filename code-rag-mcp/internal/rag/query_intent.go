package rag

import (
	"regexp"
	"strings"
)

// QueryIntent represents the analyzed intent of a search query
type QueryIntent struct {
	SearchType   string            // "definition", "usage", "implementation", "test", "example"
	EntityType   string            // "function", "class", "variable", "import", "type"
	Keywords     []string          // Extracted meaningful keywords
	Language     string            // Detected programming language
	Constraints  map[string]string // Additional constraints (file type, path, etc.)
	Confidence   float32           // Confidence in the intent analysis
}

// QueryAnalyzer analyzes search queries to understand intent
type QueryAnalyzer struct {
	// Patterns for detecting search intent
	searchTypePatterns map[string][]*regexp.Regexp
	entityTypePatterns map[string][]*regexp.Regexp
	languagePatterns   map[string][]*regexp.Regexp
}

// NewQueryAnalyzer creates a new query analyzer
func NewQueryAnalyzer() *QueryAnalyzer {
	return &QueryAnalyzer{
		searchTypePatterns: map[string][]*regexp.Regexp{
			"usage": {
				regexp.MustCompile(`(?i)(where|how).*(used|called|invoked|referenced)`),
				regexp.MustCompile(`(?i)(usage|uses|calls|references)\s+of`),
				regexp.MustCompile(`(?i)who\s+(calls|uses|invokes)`),
			},
			"implementation": {
				regexp.MustCompile(`(?i)(implement|create|write|build|make)\s+`),
				regexp.MustCompile(`(?i)how\s+to\s+(implement|create|write|build)`),
				regexp.MustCompile(`(?i)implementation\s+of`),
			},
			"definition": {
				regexp.MustCompile(`(?i)(define|definition|declaration)\s+`),
				regexp.MustCompile(`(?i)where\s+(is|are).*(defined|declared)`),
				regexp.MustCompile(`(?i)(func|function|class|struct|type|interface)\s+\w+`),
			},
			"test": {
				regexp.MustCompile(`(?i)(test|spec|mock|stub)\s+`),
				regexp.MustCompile(`(?i)test.*for\s+`),
				regexp.MustCompile(`(?i)(unit|integration|e2e)\s+test`),
			},
			"example": {
				regexp.MustCompile(`(?i)example\s+(of|for)`),
				regexp.MustCompile(`(?i)how\s+to\s+use`),
				regexp.MustCompile(`(?i)sample\s+code`),
			},
		},
		entityTypePatterns: map[string][]*regexp.Regexp{
			"function": {
				regexp.MustCompile(`(?i)(func|function|method|procedure|routine)\s+`),
				regexp.MustCompile(`(?i)\w+\s*\([^)]*\)`), // Function call pattern
			},
			"class": {
				regexp.MustCompile(`(?i)(class|struct|type)\s+`),
				regexp.MustCompile(`(?i)new\s+\w+`),
			},
			"variable": {
				regexp.MustCompile(`(?i)(var|variable|const|constant|field|property)\s+`),
				regexp.MustCompile(`(?i)(get|set)\s+\w+`),
			},
			"import": {
				regexp.MustCompile(`(?i)(import|require|include|use)\s+`),
				regexp.MustCompile(`(?i)dependency\s+`),
			},
			"interface": {
				regexp.MustCompile(`(?i)interface\s+`),
				regexp.MustCompile(`(?i)implements\s+`),
			},
		},
		languagePatterns: map[string][]*regexp.Regexp{
			"go": {
				regexp.MustCompile(`(?i)\b(go|golang)\b`),
				regexp.MustCompile(`(?i)\b(goroutine|channel|defer)\b`),
				regexp.MustCompile(`(?i)\.go\b`),
			},
			"javascript": {
				regexp.MustCompile(`(?i)\b(js|javascript|node|nodejs)\b`),
				regexp.MustCompile(`(?i)\b(async|await|promise|callback)\b`),
				regexp.MustCompile(`(?i)\.(js|jsx|mjs)\b`),
			},
			"typescript": {
				regexp.MustCompile(`(?i)\b(ts|typescript)\b`),
				regexp.MustCompile(`(?i)\b(interface|type|enum)\b`),
				regexp.MustCompile(`(?i)\.(ts|tsx)\b`),
			},
			"python": {
				regexp.MustCompile(`(?i)\b(python|py)\b`),
				regexp.MustCompile(`(?i)\b(def|class|import from)\b`),
				regexp.MustCompile(`(?i)\.py\b`),
			},
			"react": {
				regexp.MustCompile(`(?i)\b(react|component|hooks?)\b`),
				regexp.MustCompile(`(?i)\b(useState|useEffect|props)\b`),
				regexp.MustCompile(`(?i)\.(jsx|tsx)\b`),
			},
		},
	}
}

// AnalyzeQuery analyzes a search query to understand its intent
func (qa *QueryAnalyzer) AnalyzeQuery(query string) QueryIntent {
	intent := QueryIntent{
		SearchType:  "general",
		EntityType:  "any",
		Keywords:    qa.extractKeywords(query),
		Constraints: make(map[string]string),
		Confidence:  0.5,
	}
	
	// Detect search type
	for searchType, patterns := range qa.searchTypePatterns {
		for _, pattern := range patterns {
			if pattern.MatchString(query) {
				intent.SearchType = searchType
				intent.Confidence += 0.1
				break
			}
		}
	}
	
	// Detect entity type
	for entityType, patterns := range qa.entityTypePatterns {
		for _, pattern := range patterns {
			if pattern.MatchString(query) {
				intent.EntityType = entityType
				intent.Confidence += 0.1
				break
			}
		}
	}
	
	// Detect language
	for language, patterns := range qa.languagePatterns {
		for _, pattern := range patterns {
			if pattern.MatchString(query) {
				intent.Language = language
				intent.Confidence += 0.05
				break
			}
		}
	}
	
	// Extract file path constraints
	if matches := regexp.MustCompile(`(?i)in\s+(\S+\.go|\S+\.js|\S+\.py|\S+/\S+)`).FindStringSubmatch(query); len(matches) > 1 {
		intent.Constraints["path"] = matches[1]
		intent.Confidence += 0.1
	}
	
	// Extract specific function/class names
	if matches := regexp.MustCompile(`(?i)(?:func|function|class|struct|type)\s+(\w+)`).FindStringSubmatch(query); len(matches) > 1 {
		intent.Constraints["name"] = matches[1]
		intent.Confidence += 0.2
	}
	
	// Cap confidence at 1.0
	if intent.Confidence > 1.0 {
		intent.Confidence = 1.0
	}
	
	return intent
}

// extractKeywords extracts meaningful keywords from the query
func (qa *QueryAnalyzer) extractKeywords(query string) []string {
	// Remove common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"as": true, "is": true, "was": true, "are": true, "were": true,
		"been": true, "be": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "can": true,
		"where": true, "how": true, "what": true, "which": true, "who": true,
	}
	
	words := strings.Fields(strings.ToLower(query))
	keywords := []string{}
	
	for _, word := range words {
		// Remove punctuation
		word = regexp.MustCompile(`[^\w]`).ReplaceAllString(word, "")
		
		// Skip if empty or stop word
		if word == "" || stopWords[word] {
			continue
		}
		
		// Skip very short words unless they're important (like "go", "js")
		if len(word) < 2 && word != "c" && word != "r" {
			continue
		}
		
		keywords = append(keywords, word)
	}
	
	return keywords
}

// ApplyIntentToSearch applies the query intent to improve search results
func (qa *QueryAnalyzer) ApplyIntentToSearch(intent QueryIntent, results []SearchResult) []SearchResult {
	// Boost results based on intent
	for i := range results {
		boost := float32(1.0)
		
		// Boost based on search type
		switch intent.SearchType {
		case "definition":
			if results[i].Type == "function" || results[i].Type == "class" || results[i].Type == "type" {
				boost *= 1.3
			}
		case "usage":
			// Boost files that are likely to contain usage (not definitions)
			if !strings.Contains(strings.ToLower(results[i].FilePath), "test") &&
			   !strings.Contains(strings.ToLower(results[i].FilePath), "interface") {
				boost *= 1.1
			}
		case "test":
			if strings.Contains(strings.ToLower(results[i].FilePath), "test") ||
			   strings.Contains(strings.ToLower(results[i].FilePath), "spec") {
				boost *= 1.5
			}
		case "implementation":
			if results[i].Type == "function" || results[i].Type == "method" {
				boost *= 1.2
			}
		}
		
		// Boost based on entity type match
		if intent.EntityType != "any" && results[i].Type == intent.EntityType {
			boost *= 1.2
		}
		
		// Boost based on language match
		if intent.Language != "" && results[i].Language == intent.Language {
			boost *= 1.15
		}
		
		// Boost based on name match
		if name, ok := intent.Constraints["name"]; ok {
			if strings.Contains(strings.ToLower(results[i].Name), strings.ToLower(name)) {
				boost *= 2.0 // Strong boost for exact name matches
			}
		}
		
		// Boost based on path constraint
		if path, ok := intent.Constraints["path"]; ok {
			if strings.Contains(results[i].FilePath, path) {
				boost *= 1.5
			}
		}
		
		results[i].Score *= boost
	}
	
	// Re-sort by score
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
	
	return results
}