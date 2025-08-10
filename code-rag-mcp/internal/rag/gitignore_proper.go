package rag

import (
	"path/filepath"
	"strings"
	
	gitignore "github.com/denormal/go-gitignore"
)

// ProperGitIgnore uses a proper gitignore parser library
type ProperGitIgnore struct {
	matcher gitignore.GitIgnore
	basePath string
}

// NewProperGitIgnore creates a gitignore matcher using the proper library
func NewProperGitIgnore(projectPath string) (*ProperGitIgnore, error) {
	// Create a repository instance for the project path
	repository, err := gitignore.NewRepository(projectPath)
	if err != nil {
		// If we can't create a repository, create an empty matcher
		return &ProperGitIgnore{
			matcher: nil,
			basePath: projectPath,
		}, nil
	}
	
	// This will automatically load .gitignore files in the path
	matcher := repository
	
	return &ProperGitIgnore{
		matcher: matcher,
		basePath: projectPath,
	}, nil
}

// IsIgnored checks if a path should be ignored according to gitignore rules
func (g *ProperGitIgnore) IsIgnored(path string) bool {
	if g.matcher == nil {
		return false
	}
	
	// Get relative path from base
	relPath, err := filepath.Rel(g.basePath, path)
	if err != nil {
		return false
	}
	
	// Normalize path separators for gitignore matching
	relPath = filepath.ToSlash(relPath)
	
	// Check if path matches gitignore patterns
	match := g.matcher.Ignore(relPath)
	
	return match
}

// IsIgnoredDir checks if a directory should be ignored
func (g *ProperGitIgnore) IsIgnoredDir(path string) bool {
	if g.matcher == nil {
		return false
	}
	
	// Get relative path from base
	relPath, err := filepath.Rel(g.basePath, path)
	if err != nil {
		return false
	}
	
	// For directories, append / for proper gitignore matching
	relPath = filepath.ToSlash(relPath)
	if !strings.HasSuffix(relPath, "/") {
		relPath = relPath + "/"
	}
	
	return g.matcher.Ignore(relPath)
}