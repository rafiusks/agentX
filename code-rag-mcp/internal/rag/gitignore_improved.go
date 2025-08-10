package rag

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ImprovedGitIgnore represents a proper gitignore file parser
type ImprovedGitIgnore struct {
	patterns []compiledPattern
	basePath string
}

type compiledPattern struct {
	original  string
	regex     *regexp.Regexp
	isNegated bool
	isDir     bool
}

// NewImprovedGitIgnore creates a new gitignore parser with proper pattern matching
func NewImprovedGitIgnore(path string) (*ImprovedGitIgnore, error) {
	gi := &ImprovedGitIgnore{
		basePath: filepath.Dir(path),
		patterns: []compiledPattern{},
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

func (gi *ImprovedGitIgnore) loadFile(path string) error {
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
		
		pattern := compiledPattern{original: line}
		
		// Check if negated
		if strings.HasPrefix(line, "!") {
			pattern.isNegated = true
			line = line[1:]
		}
		
		// Check if directory only
		if strings.HasSuffix(line, "/") {
			pattern.isDir = true
			line = strings.TrimSuffix(line, "/")
		}
		
		// Compile the pattern to regex
		pattern.regex = gi.compilePattern(line)
		gi.patterns = append(gi.patterns, pattern)
	}
	
	return scanner.Err()
}

// compilePattern converts a gitignore pattern to a regular expression
func (gi *ImprovedGitIgnore) compilePattern(pattern string) *regexp.Regexp {
	// Escape special regex characters except * and ?
	escaped := regexp.QuoteMeta(pattern)
	escaped = strings.ReplaceAll(escaped, `\*`, ".*")
	escaped = strings.ReplaceAll(escaped, `\?`, ".")
	
	// Handle leading slash (absolute path from repo root)
	if strings.HasPrefix(pattern, "/") {
		escaped = "^" + escaped[1:]
	} else {
		// Pattern can match at any level
		escaped = "(^|/)" + escaped
	}
	
	// Add boundary at the end
	escaped = escaped + "($|/)"
	
	// Compile with case-insensitive flag for cross-platform compatibility
	regex, _ := regexp.Compile(escaped)
	return regex
}

// IsIgnored checks if a path should be ignored
func (gi *ImprovedGitIgnore) IsIgnored(path string, isDir bool) bool {
	// Get relative path from base
	relPath, err := filepath.Rel(gi.basePath, path)
	if err != nil {
		return false
	}
	
	// Normalize path separators to forward slashes
	relPath = filepath.ToSlash(relPath)
	
	// Check each component of the path
	// This is important for patterns like "node_modules/"
	pathComponents := strings.Split(relPath, "/")
	for i := range pathComponents {
		partialPath := strings.Join(pathComponents[:i+1], "/")
		
		// Check if this partial path matches any ignore pattern
		if gi.isPathIgnored(partialPath, isDir || i < len(pathComponents)-1) {
			return true
		}
	}
	
	return false
}

func (gi *ImprovedGitIgnore) isPathIgnored(path string, isDir bool) bool {
	ignored := false
	
	for _, pattern := range gi.patterns {
		// Skip directory-only patterns if checking a file
		if pattern.isDir && !isDir {
			continue
		}
		
		// Check if the pattern matches
		if pattern.regex != nil && pattern.regex.MatchString(path) {
			if pattern.isNegated {
				ignored = false
			} else {
				ignored = true
			}
		}
	}
	
	return ignored
}

// MatchesNodeModules specifically checks if a path is within node_modules
func (gi *ImprovedGitIgnore) MatchesNodeModules(path string) bool {
	// Get relative path from base
	relPath, err := filepath.Rel(gi.basePath, path)
	if err != nil {
		// If we can't get relative path, check absolute path
		relPath = path
	}
	
	// Normalize path separators
	relPath = filepath.ToSlash(relPath)
	
	// Check if path contains node_modules at any level
	return strings.Contains(relPath, "node_modules/") || 
	       strings.HasPrefix(relPath, "node_modules") ||
	       strings.Contains(relPath, "/node_modules/") ||
	       strings.HasSuffix(relPath, "/node_modules")
}