package rag

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ImprovedChunker creates semantic code chunks that preserve function/class boundaries
type ImprovedChunker struct {
	config *ChunkingConfig
}

func NewImprovedChunker(config *ChunkingConfig) *ImprovedChunker {
	return &ImprovedChunker{config: config}
}

// ChunkFile creates semantic chunks based on code structure
func (ic *ImprovedChunker) ChunkFile(filePath string) ([]Chunk, error) {
	ext := filepath.Ext(filePath)
	language := ic.detectLanguage(ext)
	
	switch language {
	case "go":
		return ic.chunkGoFile(filePath)
	case "javascript", "typescript", "jsx", "tsx":
		return ic.chunkJSFile(filePath)
	case "python":
		return ic.chunkPythonFile(filePath)
	default:
		// For other languages, use intelligent line-based chunking
		return ic.smartChunkFile(filePath, language)
	}
}

// chunkGoFile uses Go AST to create proper chunks
func (ic *ImprovedChunker) chunkGoFile(filePath string) ([]Chunk, error) {
	fset := token.NewFileSet()
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	node, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		// Fallback to smart chunking if parsing fails
		return ic.smartChunkFile(filePath, "go")
	}
	
	var chunks []Chunk
	
	// Add package documentation as a chunk
	if node.Doc != nil && len(node.Doc.Text()) > 20 {
		chunks = append(chunks, Chunk{
			Code:      fmt.Sprintf("package %s\n\n%s", node.Name.Name, node.Doc.Text()),
			Language:  "go",
			FilePath:  filePath,
			LineStart: fset.Position(node.Doc.Pos()).Line,
			LineEnd:   fset.Position(node.Doc.End()).Line,
			Type:      "package_doc",
			Name:      fmt.Sprintf("package %s", node.Name.Name),
		})
	}
	
	// Extract imports as context
	var imports []string
	for _, imp := range node.Imports {
		imports = append(imports, imp.Path.Value)
	}
	importContext := strings.Join(imports, "\n")
	
	// Process each top-level declaration
	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			// Extract function with its documentation
			chunk := ic.extractGoFunction(fset, src, d, importContext)
			if chunk != nil {
				chunks = append(chunks, *chunk)
			}
			
		case *ast.GenDecl:
			// Extract type definitions, constants, vars
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					chunk := ic.extractGoType(fset, src, d, s, importContext)
					if chunk != nil {
						chunks = append(chunks, *chunk)
					}
				}
			}
		}
	}
	
	// If no chunks were created, fall back to smart chunking
	if len(chunks) == 0 {
		return ic.smartChunkFile(filePath, "go")
	}
	
	return chunks, nil
}

// extractGoFunction extracts a complete function with context
func (ic *ImprovedChunker) extractGoFunction(fset *token.FileSet, src []byte, fn *ast.FuncDecl, imports string) *Chunk {
	start := fset.Position(fn.Pos())
	end := fset.Position(fn.End())
	
	// Get the actual code
	code := string(src[start.Offset:end.Offset])
	
	// Add function documentation if exists
	if fn.Doc != nil {
		code = fn.Doc.Text() + "\n" + code
	}
	
	// Create a meaningful name
	name := fn.Name.Name
	receiverType := ""
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		// Method receiver
		if t, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok {
			if ident, ok := t.X.(*ast.Ident); ok {
				receiverType = ident.Name
				name = fmt.Sprintf("%s.%s", ident.Name, name)
			}
		} else if ident, ok := fn.Recv.List[0].Type.(*ast.Ident); ok {
			receiverType = ident.Name
			name = fmt.Sprintf("%s.%s", ident.Name, name)
		}
	}
	
	// Build contextual code with imports and struct context
	contextualCode := ""
	
	// Add package and imports for context (helps embeddings understand dependencies)
	if imports != "" {
		contextualCode = fmt.Sprintf("// Imports:\n%s\n\n", imports)
	}
	
	// Add receiver type definition if this is a method (helps understand the struct)
	if receiverType != "" {
		contextualCode += fmt.Sprintf("// Method of %s\n", receiverType)
	}
	
	// Add function metadata
	contextualCode += fmt.Sprintf("// Function: %s\n// File: %s\n", 
		name, filepath.Base(fset.File(fn.Pos()).Name()))
	
	// Add the actual code
	contextualCode += code
	
	return &Chunk{
		Code:      contextualCode,
		Language:  "go",
		FilePath:  fset.File(fn.Pos()).Name(),
		LineStart: start.Line,
		LineEnd:   end.Line,
		Type:      "function",
		Name:      name,
	}
}

// extractGoType extracts type definitions with methods
func (ic *ImprovedChunker) extractGoType(fset *token.FileSet, src []byte, decl *ast.GenDecl, spec *ast.TypeSpec, imports string) *Chunk {
	start := fset.Position(decl.Pos())
	end := fset.Position(decl.End())
	
	code := string(src[start.Offset:end.Offset])
	
	// Add documentation
	if decl.Doc != nil {
		code = decl.Doc.Text() + "\n" + code
	}
	
	// Build contextual code with imports
	contextualCode := ""
	if imports != "" {
		contextualCode = fmt.Sprintf("// Imports:\n%s\n\n", imports)
	}
	contextualCode += fmt.Sprintf("// Type: %s\n%s", spec.Name.Name, code)
	
	return &Chunk{
		Code:      contextualCode,
		Language:  "go",
		FilePath:  fset.File(decl.Pos()).Name(),
		LineStart: start.Line,
		LineEnd:   end.Line,
		Type:      "type",
		Name:      spec.Name.Name,
	}
}

// chunkJSFile handles JavaScript/TypeScript files
func (ic *ImprovedChunker) chunkJSFile(filePath string) ([]Chunk, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	text := string(content)
	var chunks []Chunk
	
	// Regular expressions for JS/TS constructs
	// Function declarations and expressions
	funcRegex := regexp.MustCompile(`(?m)^(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\([^)]*\)\s*\{`)
	arrowRegex := regexp.MustCompile(`(?m)^(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\([^)]*\)\s*=>\s*\{`)
	classRegex := regexp.MustCompile(`(?m)^(?:export\s+)?(?:abstract\s+)?class\s+(\w+)`)
	
	// Find all functions
	for _, match := range funcRegex.FindAllStringSubmatchIndex(text, -1) {
		if chunk := ic.extractJSBlock(text, match[0], match[1], "function"); chunk != nil {
			chunk.FilePath = filePath
			chunk.Language = ic.detectLanguage(filepath.Ext(filePath))
			chunks = append(chunks, *chunk)
		}
	}
	
	// Find arrow functions
	for _, match := range arrowRegex.FindAllStringSubmatchIndex(text, -1) {
		if chunk := ic.extractJSBlock(text, match[0], match[1], "function"); chunk != nil {
			chunk.FilePath = filePath
			chunk.Language = ic.detectLanguage(filepath.Ext(filePath))
			chunks = append(chunks, *chunk)
		}
	}
	
	// Find classes
	for _, match := range classRegex.FindAllStringSubmatchIndex(text, -1) {
		if chunk := ic.extractJSBlock(text, match[0], match[1], "class"); chunk != nil {
			chunk.FilePath = filePath
			chunk.Language = ic.detectLanguage(filepath.Ext(filePath))
			chunks = append(chunks, *chunk)
		}
	}
	
	// Find React components (functional)
	componentRegex := regexp.MustCompile(`(?m)^(?:export\s+)?(?:const|function)\s+([A-Z]\w+)\s*[:=]`)
	for _, match := range componentRegex.FindAllStringSubmatchIndex(text, -1) {
		if chunk := ic.extractJSBlock(text, match[0], match[1], "component"); chunk != nil {
			chunk.FilePath = filePath
			chunk.Language = ic.detectLanguage(filepath.Ext(filePath))
			chunks = append(chunks, *chunk)
		}
	}
	
	// If no chunks found, use smart chunking
	if len(chunks) == 0 {
		return ic.smartChunkFile(filePath, ic.detectLanguage(filepath.Ext(filePath)))
	}
	
	return chunks, nil
}

// extractJSBlock extracts a complete JS/TS code block with context
func (ic *ImprovedChunker) extractJSBlock(text string, start, nameEnd int, blockType string) *Chunk {
	// Find the matching closing brace
	braceCount := 0
	inString := false
	stringChar := byte(0)
	end := start
	
	for i := start; i < len(text); i++ {
		ch := text[i]
		
		// Handle strings
		if !inString && (ch == '"' || ch == '\'' || ch == '`') {
			inString = true
			stringChar = ch
		} else if inString && ch == stringChar && (i == 0 || text[i-1] != '\\') {
			inString = false
		}
		
		if !inString {
			if ch == '{' {
				braceCount++
			} else if ch == '}' {
				braceCount--
				if braceCount == 0 {
					end = i + 1
					break
				}
			}
		}
	}
	
	if end <= start {
		return nil
	}
	
	// Extract the complete block
	code := text[start:end]
	lines := strings.Split(text[:start], "\n")
	lineStart := len(lines)
	lines = strings.Split(text[:end], "\n")
	lineEnd := len(lines)
	
	// Extract name
	name := "unknown"
	if nameEnd > start {
		nameMatch := regexp.MustCompile(`(\w+)`).FindString(text[start:nameEnd])
		if nameMatch != "" {
			name = nameMatch
		}
	}
	
	// Extract imports for context (look at beginning of file)
	imports := ""
	importRegex := regexp.MustCompile(`(?m)^import\s+.*?;|^import\s+\{[^}]+\}\s+from\s+['"][^'"]+['"];?|^const\s+\w+\s*=\s*require\([^)]+\);?`)
	importMatches := importRegex.FindAllString(text[:min(2000, len(text))], -1) // Check first 2000 chars
	if len(importMatches) > 0 {
		imports = strings.Join(importMatches, "\n")
	}
	
	// Build contextual code
	contextualCode := ""
	if imports != "" {
		contextualCode = fmt.Sprintf("// Imports:\n%s\n\n", imports)
	}
	contextualCode += fmt.Sprintf("// %s: %s\n", blockType, name)
	contextualCode += code
	
	return &Chunk{
		Code:      contextualCode,
		LineStart: lineStart,
		LineEnd:   lineEnd,
		Type:      blockType,
		Name:      name,
	}
}

// chunkPythonFile handles Python files
func (ic *ImprovedChunker) chunkPythonFile(filePath string) ([]Chunk, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	var chunks []Chunk
	lines := strings.Split(string(content), "\n")
	
	// Regular expressions for Python constructs
	classRegex := regexp.MustCompile(`^class\s+(\w+)`)
	funcRegex := regexp.MustCompile(`^def\s+(\w+)`)
	asyncRegex := regexp.MustCompile(`^async\s+def\s+(\w+)`)
	
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		
		// Check for class definition
		if match := classRegex.FindStringSubmatch(trimmed); match != nil {
			if chunk := ic.extractPythonBlock(lines, i, "class", match[1]); chunk != nil {
				chunk.FilePath = filePath
				chunk.Language = "python"
				chunks = append(chunks, *chunk)
				i = chunk.LineEnd - 1 // Skip processed lines
			}
		} else if match := funcRegex.FindStringSubmatch(trimmed); match != nil {
			// Check for function definition
			if chunk := ic.extractPythonBlock(lines, i, "function", match[1]); chunk != nil {
				chunk.FilePath = filePath
				chunk.Language = "python"
				chunks = append(chunks, *chunk)
				i = chunk.LineEnd - 1
			}
		} else if match := asyncRegex.FindStringSubmatch(trimmed); match != nil {
			// Check for async function
			if chunk := ic.extractPythonBlock(lines, i, "function", match[1]); chunk != nil {
				chunk.FilePath = filePath
				chunk.Language = "python"
				chunks = append(chunks, *chunk)
				i = chunk.LineEnd - 1
			}
		}
	}
	
	if len(chunks) == 0 {
		return ic.smartChunkFile(filePath, "python")
	}
	
	return chunks, nil
}

// extractPythonBlock extracts a Python code block based on indentation
func (ic *ImprovedChunker) extractPythonBlock(lines []string, start int, blockType, name string) *Chunk {
	if start >= len(lines) {
		return nil
	}
	
	// Get the base indentation
	baseIndent := len(lines[start]) - len(strings.TrimLeft(lines[start], " \t"))
	
	// Find the end of the block
	end := start + 1
	for end < len(lines) {
		if strings.TrimSpace(lines[end]) == "" {
			end++
			continue
		}
		
		currentIndent := len(lines[end]) - len(strings.TrimLeft(lines[end], " \t"))
		if currentIndent <= baseIndent {
			break
		}
		end++
	}
	
	// Include any decorators before the definition
	decoratorStart := start
	for decoratorStart > 0 && strings.TrimSpace(lines[decoratorStart-1]) != "" {
		if strings.HasPrefix(strings.TrimSpace(lines[decoratorStart-1]), "@") {
			decoratorStart--
		} else {
			break
		}
	}
	
	// Include docstrings
	docstringEnd := end
	if end < len(lines)-1 && (strings.Contains(lines[start+1], `"""`) || strings.Contains(lines[start+1], `'''`)) {
		for i := start + 2; i < len(lines); i++ {
			if strings.Contains(lines[i], `"""`) || strings.Contains(lines[i], `'''`) {
				docstringEnd = i + 1
				break
			}
		}
	}
	
	if docstringEnd > end {
		end = docstringEnd
	}
	
	code := strings.Join(lines[decoratorStart:end], "\n")
	
	return &Chunk{
		Code:      code,
		LineStart: decoratorStart + 1,
		LineEnd:   end,
		Type:      blockType,
		Name:      name,
	}
}

// smartChunkFile creates intelligent chunks for any language
func (ic *ImprovedChunker) smartChunkFile(filePath string, language string) ([]Chunk, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	lines := strings.Split(string(content), "\n")
	var chunks []Chunk
	
	// Look for natural boundaries
	var currentChunk []string
	chunkStart := 0
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Natural boundaries to split on
		isNaturalBoundary := false
		if language == "go" || language == "java" || language == "rust" {
			// Look for function/method signatures
			isNaturalBoundary = strings.HasPrefix(trimmed, "func ") ||
				strings.HasPrefix(trimmed, "public ") ||
				strings.HasPrefix(trimmed, "private ") ||
				strings.HasPrefix(trimmed, "fn ")
		}
		
		// Check if we should create a chunk
		shouldChunk := isNaturalBoundary && len(currentChunk) > ic.config.MinChunkSize
		
		if shouldChunk || (len(currentChunk) >= ic.config.MaxChunkSize) {
			if len(currentChunk) > 0 {
				code := strings.Join(currentChunk, "\n")
				chunks = append(chunks, Chunk{
					Code:      code,
					Language:  language,
					FilePath:  filePath,
					LineStart: chunkStart + 1,
					LineEnd:   i,
					Type:      "block",
					Name:      fmt.Sprintf("block_%d_%d", chunkStart+1, i),
				})
			}
			currentChunk = []string{line}
			chunkStart = i
		} else {
			currentChunk = append(currentChunk, line)
		}
	}
	
	// Add remaining chunk
	if len(currentChunk) > 0 {
		code := strings.Join(currentChunk, "\n")
		chunks = append(chunks, Chunk{
			Code:      code,
			Language:  language,
			FilePath:  filePath,
			LineStart: chunkStart + 1,
			LineEnd:   len(lines),
			Type:      "block",
			Name:      fmt.Sprintf("block_%d_%d", chunkStart+1, len(lines)),
		})
	}
	
	return chunks, nil
}

// detectLanguage detects language from file extension
func (ic *ImprovedChunker) detectLanguage(ext string) string {
	switch strings.ToLower(ext) {
	case ".go":
		return "go"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".jsx":
		return "jsx"
	case ".tsx":
		return "tsx"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".rs":
		return "rust"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".c", ".h":
		return "c"
	case ".cpp", ".hpp", ".cc":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".swift":
		return "swift"
	case ".kt":
		return "kotlin"
	case ".sql":
		return "sql"
	case ".sh", ".bash":
		return "bash"
	default:
		return "text"
	}
}