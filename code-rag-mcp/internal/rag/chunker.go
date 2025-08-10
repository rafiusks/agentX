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
			
			chunks = append(chunks, Chunk{
				Code:      code,
				Language:  "go",
				FilePath:  filePath,
				LineStart: start.Line,
				LineEnd:   end.Line,
				Type:      chunkType,
				Name:      x.Name.Name,
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