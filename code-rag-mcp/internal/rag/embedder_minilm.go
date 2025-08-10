package rag

import (
	"context"
	"crypto/sha256"
	"math"
	"strings"
	"unicode"
)

// MiniLMProvider implements a local embedding provider optimized for code
// It uses advanced hashing and TF-IDF-like techniques for better semantic matching
type MiniLMProvider struct {
	config     *EmbeddingConfig
	vocabulary map[string]int
	idf        map[string]float64
	dimension  int
}

// NewMiniLMProvider creates a new MiniLM-style local embedding provider
func NewMiniLMProvider(config *EmbeddingConfig) *MiniLMProvider {
	return &MiniLMProvider{
		config:     config,
		vocabulary: buildCodeVocabulary(),
		idf:        buildIDFWeights(),
		dimension:  768, // Standard dimension for compatibility
	}
}

// Embed generates embeddings for a single text
func (p *MiniLMProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return p.generateAdvancedEmbedding(text), nil
}

// EmbedBatch generates embeddings for multiple texts
func (p *MiniLMProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embeddings[i] = p.generateAdvancedEmbedding(text)
	}
	return embeddings, nil
}

// generateAdvancedEmbedding creates a sophisticated embedding using multiple techniques
func (p *MiniLMProvider) generateAdvancedEmbedding(text string) []float32 {
	embedding := make([]float32, p.dimension)
	
	// 1. Tokenize and extract features
	tokens := tokenizeCode(text)
	
	// 2. Character-level features (first 256 dimensions)
	charFeatures := p.extractCharFeatures(text)
	for i := 0; i < 256 && i < p.dimension; i++ {
		embedding[i] = charFeatures[i]
	}
	
	// 3. Token-level features with TF-IDF weighting (next 256 dimensions)
	tokenFeatures := p.extractTokenFeatures(tokens)
	for i := 0; i < 256 && i+256 < p.dimension; i++ {
		embedding[i+256] = tokenFeatures[i]
	}
	
	// 4. Semantic features based on code patterns (remaining dimensions)
	semanticFeatures := p.extractSemanticFeatures(text, tokens)
	for i := 0; i < 256 && i+512 < p.dimension; i++ {
		embedding[i+512] = semanticFeatures[i]
	}
	
	// 5. Normalize the embedding
	return normalize(embedding)
}

// extractCharFeatures creates character-level features
func (p *MiniLMProvider) extractCharFeatures(text string) []float32 {
	features := make([]float32, 256)
	
	// Character frequency distribution
	charCounts := make(map[rune]int)
	totalChars := 0
	for _, ch := range text {
		charCounts[ch]++
		totalChars++
	}
	
	// Hash-based distribution across dimensions
	for ch, count := range charCounts {
		freq := float32(count) / float32(totalChars)
		// Use multiple hash functions for better distribution
		for h := 0; h < 4; h++ {
			hash := hashChar(ch, h)
			idx := hash % 256
			features[idx] += freq * (1.0 / 4.0)
		}
	}
	
	return features
}

// extractTokenFeatures creates token-level features with TF-IDF weighting
func (p *MiniLMProvider) extractTokenFeatures(tokens []string) []float32 {
	features := make([]float32, 256)
	
	// Calculate term frequency
	tf := make(map[string]float64)
	for _, token := range tokens {
		tf[token]++
	}
	for token := range tf {
		tf[token] /= float64(len(tokens))
	}
	
	// Apply TF-IDF weighting
	for token, freq := range tf {
		idf := 1.0
		if idfValue, exists := p.idf[token]; exists {
			idf = idfValue
		}
		
		tfidf := freq * idf
		
		// Hash token to multiple dimensions
		for h := 0; h < 4; h++ {
			hash := hashString(token, h)
			idx := hash % 256
			features[idx] += float32(tfidf) * (1.0 / 4.0)
		}
	}
	
	return features
}

// extractSemanticFeatures extracts code-specific semantic features
func (p *MiniLMProvider) extractSemanticFeatures(text string, tokens []string) []float32 {
	features := make([]float32, 256)
	
	// Code pattern detection
	patterns := detectCodePatterns(text)
	
	// Function/method detection
	if patterns["has_function"] {
		features[0] = 1.0
	}
	if patterns["has_class"] {
		features[1] = 1.0
	}
	if patterns["has_import"] {
		features[2] = 1.0
	}
	if patterns["has_loop"] {
		features[3] = 1.0
	}
	if patterns["has_condition"] {
		features[4] = 1.0
	}
	if patterns["has_comment"] {
		features[5] = 1.0
	}
	
	// Language-specific features
	langFeatures := detectLanguageFeatures(text)
	for i, feat := range langFeatures {
		if i+10 < 256 {
			features[i+10] = feat
		}
	}
	
	// N-gram features
	ngrams := extractNgrams(tokens, 2)
	for ngram := range ngrams {
		hash := hashString(ngram, 0)
		idx := (hash % 200) + 50
		features[idx] = float32(math.Min(float64(features[idx]+0.1), 1.0))
	}
	
	return features
}

// tokenizeCode tokenizes code text
func tokenizeCode(text string) []string {
	var tokens []string
	var current []rune
	
	for _, ch := range text {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' {
			current = append(current, ch)
		} else {
			if len(current) > 0 {
				tokens = append(tokens, strings.ToLower(string(current)))
				current = []rune{}
			}
			// Also add special characters as tokens
			if !unicode.IsSpace(ch) {
				tokens = append(tokens, string(ch))
			}
		}
	}
	
	if len(current) > 0 {
		tokens = append(tokens, strings.ToLower(string(current)))
	}
	
	return tokens
}

// detectCodePatterns detects common code patterns
func detectCodePatterns(text string) map[string]bool {
	patterns := make(map[string]bool)
	lower := strings.ToLower(text)
	
	// Function patterns
	patterns["has_function"] = strings.Contains(lower, "func") ||
		strings.Contains(lower, "function") ||
		strings.Contains(lower, "def ")
	
	// Class patterns
	patterns["has_class"] = strings.Contains(lower, "class ") ||
		strings.Contains(lower, "struct ") ||
		strings.Contains(lower, "type ")
	
	// Import patterns
	patterns["has_import"] = strings.Contains(lower, "import ") ||
		strings.Contains(lower, "require") ||
		strings.Contains(lower, "include")
	
	// Loop patterns
	patterns["has_loop"] = strings.Contains(lower, "for ") ||
		strings.Contains(lower, "while ") ||
		strings.Contains(lower, "foreach")
	
	// Condition patterns
	patterns["has_condition"] = strings.Contains(lower, "if ") ||
		strings.Contains(lower, "else ") ||
		strings.Contains(lower, "switch ")
	
	// Comment patterns
	patterns["has_comment"] = strings.Contains(text, "//") ||
		strings.Contains(text, "/*") ||
		strings.Contains(text, "#")
	
	return patterns
}

// detectLanguageFeatures detects language-specific features
func detectLanguageFeatures(text string) []float32 {
	features := make([]float32, 30)
	
	// Go features
	if strings.Contains(text, "package ") || strings.Contains(text, "func (") {
		features[0] = 1.0
	}
	
	// JavaScript/TypeScript features
	if strings.Contains(text, "const ") || strings.Contains(text, "=>") ||
		strings.Contains(text, "async ") || strings.Contains(text, "await ") {
		features[1] = 1.0
	}
	
	// Python features
	if strings.Contains(text, "def ") || strings.Contains(text, "self.") ||
		strings.Contains(text, "__init__") {
		features[2] = 1.0
	}
	
	// Java features
	if strings.Contains(text, "public class") || strings.Contains(text, "private ") ||
		strings.Contains(text, "void ") {
		features[3] = 1.0
	}
	
	// React/JSX features
	if strings.Contains(text, "useState") || strings.Contains(text, "useEffect") ||
		strings.Contains(text, "<div") || strings.Contains(text, "props.") {
		features[4] = 1.0
	}
	
	return features
}

// extractNgrams extracts n-grams from tokens
func extractNgrams(tokens []string, n int) map[string]bool {
	ngrams := make(map[string]bool)
	for i := 0; i <= len(tokens)-n; i++ {
		ngram := strings.Join(tokens[i:i+n], "_")
		ngrams[ngram] = true
	}
	return ngrams
}

// hashChar hashes a character with a seed
func hashChar(ch rune, seed int) int {
	h := sha256.New()
	h.Write([]byte{byte(ch), byte(seed)})
	sum := h.Sum(nil)
	return int(sum[0])<<8 | int(sum[1])
}

// hashString hashes a string with a seed
func hashString(s string, seed int) int {
	h := sha256.New()
	h.Write([]byte(s))
	h.Write([]byte{byte(seed)})
	sum := h.Sum(nil)
	return int(sum[0])<<16 | int(sum[1])<<8 | int(sum[2])
}

// normalize normalizes an embedding vector
func normalize(embedding []float32) []float32 {
	var sum float32
	for _, v := range embedding {
		sum += v * v
	}
	
	if sum > 0 {
		norm := float32(1.0 / math.Sqrt(float64(sum)))
		for i := range embedding {
			embedding[i] *= norm
		}
	}
	
	return embedding
}

// buildCodeVocabulary builds a vocabulary of common code terms
func buildCodeVocabulary() map[string]int {
	vocab := make(map[string]int)
	
	// Common programming keywords
	keywords := []string{
		"function", "func", "def", "class", "struct", "type", "interface",
		"import", "require", "include", "from", "export", "module",
		"if", "else", "elif", "switch", "case", "for", "while", "do",
		"return", "break", "continue", "throw", "try", "catch", "finally",
		"const", "let", "var", "int", "string", "bool", "float", "double",
		"public", "private", "protected", "static", "async", "await",
		"new", "delete", "null", "nil", "none", "undefined", "void",
		"true", "false", "self", "this", "super", "extends", "implements",
	}
	
	for i, word := range keywords {
		vocab[word] = i
	}
	
	return vocab
}

// buildIDFWeights builds IDF weights for common terms
func buildIDFWeights() map[string]float64 {
	idf := make(map[string]float64)
	
	// Common terms get lower weights
	commonTerms := []string{
		"the", "a", "an", "and", "or", "but", "in", "on", "at", "to", "for",
		"i", "j", "k", "x", "y", "z", "n", "m", "tmp", "temp", "data", "value",
	}
	
	for _, term := range commonTerms {
		idf[term] = 0.1
	}
	
	// Important code terms get higher weights
	importantTerms := []string{
		"function", "class", "method", "interface", "api", "handler",
		"service", "controller", "model", "view", "component",
		"database", "query", "request", "response", "error", "exception",
	}
	
	for _, term := range importantTerms {
		idf[term] = 2.0
	}
	
	// Default weight for unknown terms
	// (handled by checking if term exists in map)
	
	return idf
}