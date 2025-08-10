package rag

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type Chunker struct {
	config *ChunkingConfig
}

type Chunk struct {
	Code       string
	Language   string
	FilePath   string
	LineStart  int
	LineEnd    int
	Type       string // "function", "class", "method", "module"
	Name       string
	Repository string
	// Enhanced symbol extraction
	Symbols    []string // Extracted function/class/variable names
	Imports    []string // Import statements
	Signatures string   // Function/method signatures
	// Hierarchical context
	FileContext   string // Package declaration and imports
	ParentContext string // Parent class/struct for methods
	SiblingContext string // Related methods in same class
}

func NewChunker(config *ChunkingConfig) *Chunker {
	return &Chunker{
		config: config,
	}
}

func (c *Chunker) ChunkFile(filePath string) ([]Chunk, error) {
	ext := filepath.Ext(filePath)
	language := c.detectLanguage(ext)
	
	if !c.isSupported(language) {
		return c.fallbackChunking(filePath, language)
	}
	
	switch language {
	case "go":
		return c.chunkGoFile(filePath)
	case "javascript", "typescript":
		return c.chunkJSFile(filePath)
	case "python":
		return c.chunkPythonFile(filePath)
	default:
		return c.fallbackChunking(filePath, language)
	}
}

func (c *Chunker) chunkGoFile(filePath string) ([]Chunk, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return c.fallbackChunking(filePath, "go")
	}
	
	var chunks []Chunk
	
	// Extract file-level context (package and imports)
	fileContext := extractGoFileContext(node, fset, filePath)
	
	// Extract package-level documentation
	if node.Doc != nil {
		chunks = append(chunks, Chunk{
			Code:      node.Doc.Text(),
			Language:  "go",
			FilePath:  filePath,
			LineStart: fset.Position(node.Doc.Pos()).Line,
			LineEnd:   fset.Position(node.Doc.End()).Line,
			Type:      "module",
			Name:      node.Name.Name,
		})
	}
	
	// Extract functions and methods
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			start := fset.Position(x.Pos())
			end := fset.Position(x.End())
			
			// Read the actual code
			code, err := c.readLines(filePath, start.Line, end.Line)
			if err != nil {
				return true
			}
			
			chunkType := "function"
			if x.Recv != nil {
				chunkType = "method"
			}
			
			// Extract symbols and signature
			symbols := extractGoSymbols(x)
			signature := extractGoSignature(x)
			
			// Extract parent context for methods
			parentContext := ""
			if x.Recv != nil {
				parentContext = extractMethodParentContext(x, node, fset, filePath)
			}
			
			chunks = append(chunks, Chunk{
				Code:          code,
				Language:      "go",
				FilePath:      filePath,
				LineStart:     start.Line,
				LineEnd:       end.Line,
				Type:          chunkType,
				Name:          x.Name.Name,
				Symbols:       symbols,
				Signatures:    signature,
				FileContext:   fileContext,
				ParentContext: parentContext,
			})
			
		case *ast.GenDecl:
			// Handle type declarations
			if x.Tok == token.TYPE {
				for _, spec := range x.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						start := fset.Position(typeSpec.Pos())
						end := fset.Position(typeSpec.End())
						
						code, err := c.readLines(filePath, start.Line, end.Line)
						if err != nil {
							continue
						}
						
						chunks = append(chunks, Chunk{
							Code:      code,
							Language:  "go",
							FilePath:  filePath,
							LineStart: start.Line,
							LineEnd:   end.Line,
							Type:      "type",
							Name:      typeSpec.Name.Name,
						})
					}
				}
			}
		}
		return true
	})
	
	return chunks, nil
}

func (c *Chunker) chunkJSFile(filePath string) ([]Chunk, error) {
	// Simplified JS/TS chunking - would use a proper parser in production
	return c.fallbackChunking(filePath, "javascript")
}

func (c *Chunker) chunkPythonFile(filePath string) ([]Chunk, error) {
	// Simplified Python chunking - would use ast module in production
	return c.fallbackChunking(filePath, "python")
}

func (c *Chunker) fallbackChunking(filePath string, language string) ([]Chunk, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	lines := strings.Split(string(content), "\n")
	var chunks []Chunk
	
	// Sliding window chunking
	chunkSize := c.config.MaxChunkSize
	overlap := c.config.ChunkOverlap
	
	for i := 0; i < len(lines); i += chunkSize - overlap {
		end := i + chunkSize
		if end > len(lines) {
			end = len(lines)
		}
		
		chunkLines := lines[i:end]
		if len(strings.TrimSpace(strings.Join(chunkLines, "\n"))) < c.config.MinChunkSize {
			continue
		}
		
		chunks = append(chunks, Chunk{
			Code:      strings.Join(chunkLines, "\n"),
			Language:  language,
			FilePath:  filePath,
			LineStart: i + 1,
			LineEnd:   end,
			Type:      "block",
			Name:      fmt.Sprintf("lines_%d_%d", i+1, end),
		})
	}
	
	return chunks, nil
}

func (c *Chunker) ExtractDependencies(filePath string) (*Dependencies, error) {
	ext := filepath.Ext(filePath)
	language := c.detectLanguage(ext)
	
	switch language {
	case "go":
		return c.extractGoDependencies(filePath)
	default:
		return &Dependencies{}, nil
	}
}

func (c *Chunker) extractGoDependencies(filePath string) (*Dependencies, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	
	deps := &Dependencies{
		Imports:   []string{},
		Exports:   []string{},
		Functions: []string{},
	}
	
	// Extract imports
	for _, imp := range node.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		deps.Imports = append(deps.Imports, path)
	}
	
	// Extract exported functions and types
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if ast.IsExported(x.Name.Name) {
				deps.Exports = append(deps.Exports, x.Name.Name)
			}
			deps.Functions = append(deps.Functions, x.Name.Name)
			
		case *ast.GenDecl:
			if x.Tok == token.TYPE {
				for _, spec := range x.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if ast.IsExported(typeSpec.Name.Name) {
							deps.Exports = append(deps.Exports, typeSpec.Name.Name)
						}
					}
				}
			}
		}
		return true
	})
	
	return deps, nil
}

func (c *Chunker) readLines(filePath string, startLine, endLine int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	var lines []string
	lineNum := 1
	
	for scanner.Scan() {
		if lineNum >= startLine && lineNum <= endLine {
			lines = append(lines, scanner.Text())
		}
		lineNum++
		if lineNum > endLine {
			break
		}
	}
	
	return strings.Join(lines, "\n"), scanner.Err()
}

func (c *Chunker) detectLanguage(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".js":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	default:
		return "unknown"
	}
}

func (c *Chunker) isSupported(language string) bool {
	for _, lang := range c.config.Languages {
		if lang == language {
			return true
		}
	}
	return false
}

// extractGoSymbols extracts symbols from a Go function declaration
func extractGoSymbols(fn *ast.FuncDecl) []string {
	symbols := []string{fn.Name.Name}
	
	// Add receiver type if it's a method
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		for _, field := range fn.Recv.List {
			if starExpr, ok := field.Type.(*ast.StarExpr); ok {
				if ident, ok := starExpr.X.(*ast.Ident); ok {
					symbols = append(symbols, ident.Name)
				}
			} else if ident, ok := field.Type.(*ast.Ident); ok {
				symbols = append(symbols, ident.Name)
			}
		}
	}
	
	// Add parameter types
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			for _, name := range field.Names {
				symbols = append(symbols, name.Name)
			}
		}
	}
	
	return symbols
}

// extractGoSignature extracts the function signature
func extractGoSignature(fn *ast.FuncDecl) string {
	signature := fn.Name.Name + "("
	
	// Add parameters
	if fn.Type.Params != nil && len(fn.Type.Params.List) > 0 {
		params := []string{}
		for _, field := range fn.Type.Params.List {
			paramType := extractTypeString(field.Type)
			if len(field.Names) > 0 {
				for _, name := range field.Names {
					params = append(params, name.Name+" "+paramType)
				}
			} else {
				params = append(params, paramType)
			}
		}
		signature += strings.Join(params, ", ")
	}
	signature += ")"
	
	// Add return types
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		results := []string{}
		for _, field := range fn.Type.Results.List {
			results = append(results, extractTypeString(field.Type))
		}
		if len(results) == 1 {
			signature += " " + results[0]
		} else {
			signature += " (" + strings.Join(results, ", ") + ")"
		}
	}
	
	return signature
}

// extractTypeString extracts type as string from AST
func extractTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + extractTypeString(t.X)
	case *ast.ArrayType:
		return "[]" + extractTypeString(t.Elt)
	case *ast.MapType:
		return "map[" + extractTypeString(t.Key) + "]" + extractTypeString(t.Value)
	case *ast.SelectorExpr:
		return extractTypeString(t.X) + "." + t.Sel.Name
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return "interface{}"
	}
}

// extractGoFileContext extracts package declaration and imports
func extractGoFileContext(node *ast.File, fset *token.FileSet, filePath string) string {
	context := "package " + node.Name.Name + "\n\n"
	
	// Add imports
	if len(node.Imports) > 0 {
		context += "// Imports:\n"
		for _, imp := range node.Imports {
			if imp.Name != nil {
				context += imp.Name.Name + " "
			}
			context += strings.Trim(imp.Path.Value, "\"") + "\n"
		}
	}
	
	return context
}

// extractMethodParentContext extracts the parent type definition for a method
func extractMethodParentContext(method *ast.FuncDecl, file *ast.File, fset *token.FileSet, filePath string) string {
	if method.Recv == nil || len(method.Recv.List) == 0 {
		return ""
	}
	
	// Get receiver type name
	receiverType := ""
	field := method.Recv.List[0]
	switch t := field.Type.(type) {
	case *ast.Ident:
		receiverType = t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			receiverType = ident.Name
		}
	}
	
	if receiverType == "" {
		return ""
	}
	
	// Find the type declaration
	context := ""
	ast.Inspect(file, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.Name == receiverType {
					context = fmt.Sprintf("// Type: %s\n", receiverType)
					
					// Add type definition summary
					switch typeSpec.Type.(type) {
					case *ast.StructType:
						context += fmt.Sprintf("%s is a struct", receiverType)
						// Could extract field names here if needed
					case *ast.InterfaceType:
						context += fmt.Sprintf("%s is an interface", receiverType)
					default:
						context += fmt.Sprintf("%s is a type alias", receiverType)
					}
					return false
				}
			}
		}
		return true
	})
	
	return context
}