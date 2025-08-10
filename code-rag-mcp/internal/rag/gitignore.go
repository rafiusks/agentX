package rag

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// GitIgnore represents a gitignore file parser
type GitIgnore struct {
	patterns []ignorePattern
	basePath string
}

type ignorePattern struct {
	pattern   string
	isNegated bool
	isDir     bool
}

// NewGitIgnore creates a new gitignore parser
func NewGitIgnore(path string) (*GitIgnore, error) {
	gi := &GitIgnore{
		basePath: filepath.Dir(path),
		patterns: []ignorePattern{},
	}
	
	// Read .gitignore
	if err := gi.loadFile(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	
	// Also check for .code-ragignore
	codeRagIgnore := filepath.Join(filepath.Dir(path), ".code-ragignore")
	if err := gi.loadFile(codeRagIgnore); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	
	return gi, nil
}

func (gi *GitIgnore) loadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		pattern := ignorePattern{pattern: line}
		
		// Check if negated
		if strings.HasPrefix(line, "!") {
			pattern.isNegated = true
			pattern.pattern = line[1:]
		}
		
		// Check if directory only
		if strings.HasSuffix(pattern.pattern, "/") {
			pattern.isDir = true
			pattern.pattern = strings.TrimSuffix(pattern.pattern, "/")
		}
		
		gi.patterns = append(gi.patterns, pattern)
	}
	
	return scanner.Err()
}

// IsIgnored checks if a path should be ignored
func (gi *GitIgnore) IsIgnored(path string, isDir bool) bool {
	// Get relative path from base
	relPath, err := filepath.Rel(gi.basePath, path)
	if err != nil {
		return false
	}
	
	// Normalize path separators
	relPath = filepath.ToSlash(relPath)
	
	ignored := false
	for _, pattern := range gi.patterns {
		if pattern.isDir && !isDir {
			continue
		}
		
		matched := gi.matchPattern(relPath, pattern.pattern)
		if matched {
			if pattern.isNegated {
				ignored = false
			} else {
				ignored = true
			}
		}
	}
	
	return ignored
}

func (gi *GitIgnore) matchPattern(path, pattern string) bool {
	// Simple glob matching - this is a simplified version
	// A full implementation would need more complex pattern matching
	
	// Handle ** for any number of directories
	pattern = strings.ReplaceAll(pattern, "**", "†")
	pattern = strings.ReplaceAll(pattern, "*", "‡")
	pattern = strings.ReplaceAll(pattern, "†", ".*")
	pattern = strings.ReplaceAll(pattern, "‡", "[^/]*")
	
	// If pattern doesn't start with /, it can match at any level
	if !strings.HasPrefix(pattern, "/") {
		pattern = "(^|.*/)" + pattern
	} else {
		pattern = "^" + strings.TrimPrefix(pattern, "/")
	}
	
	// Add $ to match end of string if pattern doesn't end with /
	if !strings.HasSuffix(pattern, "/") {
		pattern = pattern + "($|/)"
	}
	
	// Use simple string matching for now
	// In production, use a proper regex or glob library
	return matchSimplePattern(path, pattern)
}

func matchSimplePattern(path, pattern string) bool {
	// This is a very simplified pattern matcher
	// For production, use a proper gitignore library like go-git/go-git
	
	// Handle some common cases
	if strings.Contains(pattern, "*") {
		// Convert to simple prefix/suffix match
		if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
			return strings.Contains(path, strings.Trim(pattern, "*"))
		} else if strings.HasPrefix(pattern, "*") {
			return strings.HasSuffix(path, strings.TrimPrefix(pattern, "*"))
		} else if strings.HasSuffix(pattern, "*") {
			return strings.HasPrefix(path, strings.TrimSuffix(pattern, "*"))
		}
	}
	
	// Direct match or prefix match for directories
	return path == pattern || strings.HasPrefix(path, pattern+"/")
}