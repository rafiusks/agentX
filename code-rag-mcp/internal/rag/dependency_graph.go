package rag

import (
	"bufio"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// DependencyGraph tracks file dependencies for invalidation
type DependencyGraph struct {
	mu sync.RWMutex
	// Map from file to files it imports
	imports map[string][]string
	// Map from file to files that import it (reverse index)
	importedBy map[string][]string
	// Map from file to symbols it exports
	exports map[string][]string
	// Map from symbol to files that export it
	symbolToFile map[string][]string
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		imports:      make(map[string][]string),
		importedBy:   make(map[string][]string),
		exports:      make(map[string][]string),
		symbolToFile: make(map[string][]string),
	}
}

// AnalyzeFile extracts dependencies from a file
func (dg *DependencyGraph) AnalyzeFile(filePath string, content string) {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	// Clear old dependencies for this file
	dg.clearFileDependencies(filePath)

	language := detectLanguage(filePath)
	
	switch language {
	case "go":
		dg.analyzeGoFile(filePath, content)
	case "javascript", "typescript":
		dg.analyzeJSFile(filePath, content)
	case "python":
		dg.analyzePythonFile(filePath, content)
	case "java":
		dg.analyzeJavaFile(filePath, content)
	}
}

// analyzeGoFile extracts Go imports and exports
func (dg *DependencyGraph) analyzeGoFile(filePath string, content string) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	
	// Extract imports
	importRe := regexp.MustCompile(`import\s+(?:"([^"]+)"|` + "`" + `([^` + "`" + `]+)` + "`" + `)`)
	multiImportRe := regexp.MustCompile(`import\s+\(`)
	
	inImportBlock := false
	for scanner.Scan() {
		line := scanner.Text()
		
		// Check for import block
		if multiImportRe.MatchString(line) {
			inImportBlock = true
			continue
		}
		
		if inImportBlock {
			if strings.Contains(line, ")") {
				inImportBlock = false
				continue
			}
			// Extract import from block
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "//") {
				importPath := extractImportPath(line)
				if importPath != "" && !strings.Contains(importPath, "/vendor/") {
					dg.addImport(filePath, importPath)
				}
			}
		} else {
			// Single line import
			if matches := importRe.FindStringSubmatch(line); len(matches) > 0 {
				importPath := matches[1]
				if importPath == "" {
					importPath = matches[2]
				}
				if importPath != "" && !strings.Contains(importPath, "/vendor/") {
					dg.addImport(filePath, importPath)
				}
			}
		}
		
		// Extract exported functions and types (start with capital letter in Go)
		if strings.HasPrefix(line, "func ") || strings.HasPrefix(line, "type ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[1]
				// Remove receiver for methods
				if strings.Contains(name, "(") {
					continue
				}
				// Check if exported (starts with capital)
				if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
					dg.addExport(filePath, name)
				}
			}
		}
	}
}

// analyzeJSFile extracts JavaScript/TypeScript imports and exports
func (dg *DependencyGraph) analyzeJSFile(filePath string, content string) {
	// ES6 imports
	importRe := regexp.MustCompile(`import\s+.*?\s+from\s+['"]([^'"]+)['"]`)
	requireRe := regexp.MustCompile(`require\(['"]([^'"]+)['"]\)`)
	
	// ES6 exports
	exportRe := regexp.MustCompile(`export\s+(?:default\s+)?(?:function|class|const|let|var)\s+(\w+)`)
	
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		
		// Extract imports
		if matches := importRe.FindStringSubmatch(line); len(matches) > 1 {
			importPath := resolveJSImport(filePath, matches[1])
			dg.addImport(filePath, importPath)
		}
		
		if matches := requireRe.FindStringSubmatch(line); len(matches) > 1 {
			importPath := resolveJSImport(filePath, matches[1])
			dg.addImport(filePath, importPath)
		}
		
		// Extract exports
		if matches := exportRe.FindStringSubmatch(line); len(matches) > 1 {
			dg.addExport(filePath, matches[1])
		}
	}
}

// analyzePythonFile extracts Python imports
func (dg *DependencyGraph) analyzePythonFile(filePath string, content string) {
	importRe := regexp.MustCompile(`(?:from\s+(\S+)\s+)?import\s+(.+)`)
	
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if matches := importRe.FindStringSubmatch(line); len(matches) > 0 {
			if matches[1] != "" {
				// from X import Y
				module := matches[1]
				if !strings.HasPrefix(module, ".") {
					dg.addImport(filePath, module)
				}
			} else {
				// import X, Y, Z
				imports := strings.Split(matches[2], ",")
				for _, imp := range imports {
					imp = strings.TrimSpace(imp)
					if imp != "" && !strings.HasPrefix(imp, ".") {
						dg.addImport(filePath, imp)
					}
				}
			}
		}
		
		// Extract function and class definitions
		if strings.HasPrefix(line, "def ") || strings.HasPrefix(line, "class ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := strings.TrimSuffix(parts[1], "(")
				name = strings.TrimSuffix(name, ":")
				if !strings.HasPrefix(name, "_") {
					dg.addExport(filePath, name)
				}
			}
		}
	}
}

// analyzeJavaFile extracts Java imports
func (dg *DependencyGraph) analyzeJavaFile(filePath string, content string) {
	importRe := regexp.MustCompile(`import\s+(?:static\s+)?([^;]+);`)
	
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		
		if matches := importRe.FindStringSubmatch(line); len(matches) > 1 {
			importPath := matches[1]
			if !strings.HasPrefix(importPath, "java.") {
				dg.addImport(filePath, importPath)
			}
		}
		
		// Extract public class/interface
		if strings.Contains(line, "public class") || strings.Contains(line, "public interface") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if (part == "class" || part == "interface") && i+1 < len(parts) {
					className := parts[i+1]
					// Remove generic parameters
					if idx := strings.Index(className, "<"); idx > 0 {
						className = className[:idx]
					}
					dg.addExport(filePath, className)
					break
				}
			}
		}
	}
}

// GetDependents returns all files that depend on the given file
func (dg *DependencyGraph) GetDependents(filePath string) []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	dependents := make(map[string]bool)
	
	// Direct importers
	for _, importer := range dg.importedBy[filePath] {
		dependents[importer] = true
	}
	
	// Files that import symbols exported by this file
	if _, ok := dg.exports[filePath]; ok {
		// Find all files that might import from this file
		// This is a simplification - in reality we'd need to parse usage
		for file, imports := range dg.imports {
			for _, imp := range imports {
				if strings.Contains(imp, filepath.Base(filePath)) {
					dependents[file] = true
				}
			}
		}
	}
	
	result := make([]string, 0, len(dependents))
	for dep := range dependents {
		result = append(result, dep)
	}
	return result
}

// GetTransitiveDependents returns all files that transitively depend on the given file
func (dg *DependencyGraph) GetTransitiveDependents(filePath string) []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	visited := make(map[string]bool)
	queue := []string{filePath}
	
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		
		if visited[current] {
			continue
		}
		visited[current] = true
		
		// Add direct dependents to queue
		for _, dep := range dg.importedBy[current] {
			if !visited[dep] {
				queue = append(queue, dep)
			}
		}
	}
	
	// Remove the original file from results
	delete(visited, filePath)
	
	result := make([]string, 0, len(visited))
	for dep := range visited {
		result = append(result, dep)
	}
	return result
}

// clearFileDependencies removes all dependency information for a file
func (dg *DependencyGraph) clearFileDependencies(filePath string) {
	// Remove from imports
	if imports, ok := dg.imports[filePath]; ok {
		for _, imported := range imports {
			// Remove from reverse index
			if importers, exists := dg.importedBy[imported]; exists {
				dg.importedBy[imported] = removeFromSlice(importers, filePath)
			}
		}
		delete(dg.imports, filePath)
	}
	
	// Remove exports
	if exports, ok := dg.exports[filePath]; ok {
		for _, sym := range exports {
			if files, exists := dg.symbolToFile[sym]; exists {
				dg.symbolToFile[sym] = removeFromSlice(files, filePath)
			}
		}
		delete(dg.exports, filePath)
	}
}

// addImport adds an import relationship
func (dg *DependencyGraph) addImport(importer, imported string) {
	// Resolve to actual file path if possible
	imported = resolveImportPath(importer, imported)
	
	if !contains(dg.imports[importer], imported) {
		dg.imports[importer] = append(dg.imports[importer], imported)
	}
	
	if !contains(dg.importedBy[imported], importer) {
		dg.importedBy[imported] = append(dg.importedBy[imported], importer)
	}
}

// addExport adds an exported symbol
func (dg *DependencyGraph) addExport(filePath, symbol string) {
	if !contains(dg.exports[filePath], symbol) {
		dg.exports[filePath] = append(dg.exports[filePath], symbol)
	}
	
	if !contains(dg.symbolToFile[symbol], filePath) {
		dg.symbolToFile[symbol] = append(dg.symbolToFile[symbol], filePath)
	}
}

// Helper functions

func detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".rs":
		return "rust"
	case ".cpp", ".cc", ".cxx", ".c++":
		return "cpp"
	case ".c", ".h":
		return "c"
	case ".cs":
		return "csharp"
	case ".php":
		return "php"
	default:
		return "unknown"
	}
}

func extractImportPath(line string) string {
	line = strings.TrimSpace(line)
	// Remove quotes
	line = strings.Trim(line, `"`)
	line = strings.Trim(line, "`")
	// Remove alias if present
	if idx := strings.Index(line, " "); idx > 0 {
		return line[idx+1:]
	}
	return line
}

func resolveImportPath(currentFile, importPath string) string {
	// For relative imports, try to resolve to absolute path
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		dir := filepath.Dir(currentFile)
		resolved := filepath.Join(dir, importPath)
		return filepath.Clean(resolved)
	}
	return importPath
}

func resolveJSImport(currentFile, importPath string) string {
	// Handle relative imports
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		dir := filepath.Dir(currentFile)
		resolved := filepath.Join(dir, importPath)
		// Add common extensions if missing
		if filepath.Ext(resolved) == "" {
			// Try common extensions
			for _, ext := range []string{".js", ".ts", ".jsx", ".tsx", "/index.js", "/index.ts"} {
				if _, err := filepath.Abs(resolved + ext); err == nil {
					return resolved + ext
				}
			}
		}
		return filepath.Clean(resolved)
	}
	// Node modules or absolute imports
	return importPath
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// SaveToFile persists the dependency graph
func (dg *DependencyGraph) SaveToFile(path string) error {
	// Implementation for persistence
	return nil
}

// LoadFromFile loads a persisted dependency graph
func (dg *DependencyGraph) LoadFromFile(path string) error {
	// Implementation for loading
	return nil
}