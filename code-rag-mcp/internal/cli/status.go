package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ProjectInfo struct {
	Path         string
	Name         string
	FilesIndexed int
	LastIndexed  time.Time
	Size         int64
}

func (c *CLI) getProjectStatus() []ProjectInfo {
	var projects []ProjectInfo
	
	// Read from actual index metadata
	for _, projectPath := range c.config.ProjectsIndexed {
		info := c.getProjectInfo(projectPath)
		projects = append(projects, info)
	}
	
	return projects
}

func (c *CLI) getProjectInfo(projectPath string) ProjectInfo {
	info := ProjectInfo{
		Path: projectPath,
		Name: filepath.Base(projectPath),
	}
	
	// Check for index metadata file
	metadataPath := filepath.Join(os.Getenv("HOME"), ".code-rag", "index", 
		strings.ReplaceAll(projectPath, "/", "_") + ".meta")
	
	if _, err := os.ReadFile(metadataPath); err == nil {
		// Parse metadata (in real implementation)
		// For now, use placeholders
		info.FilesIndexed = 0
		info.LastIndexed = time.Now()
	}
	
	// Get directory size and file count
	filepath.Walk(projectPath, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		// Count relevant files
		ext := filepath.Ext(path)
		if isCodeFile(ext) {
			info.FilesIndexed++
			info.Size += fileInfo.Size()
		}
		
		return nil
	})
	
	return info
}

func (c *CLI) showDetailedProjectStatus() {
	fmt.Println("\n📁 Indexed Projects")
	fmt.Println(strings.Repeat("─", 60))
	
	projects := c.getProjectStatus()
	
	if len(projects) == 0 {
		fmt.Println("No projects indexed yet.")
		fmt.Println("\nTo index a project:")
		fmt.Println("  code-rag index /path/to/project")
		fmt.Println("  code-rag index .  (current directory)")
		return
	}
	
	for _, proj := range projects {
		fmt.Printf("\n📂 %s\n", proj.Name)
		fmt.Printf("   Path: %s\n", proj.Path)
		
		if proj.FilesIndexed > 0 {
			fmt.Printf("   Files: %d indexed\n", proj.FilesIndexed)
			fmt.Printf("   Size: %s\n", formatSize(proj.Size))
			fmt.Printf("   Last indexed: %s\n", formatTime(proj.LastIndexed))
		} else {
			fmt.Printf("   Status: Not indexed (directory not found or empty)\n")
		}
	}
	
	fmt.Println("\n💡 Tips:")
	fmt.Println("  • Re-index a project: code-rag index /path/to/project")
	fmt.Println("  • Search across all: code-rag search 'your query'")
	fmt.Println("  • Search specific project: code-rag search 'query' --project name")
}

func (c *CLI) getIndexStats() (totalFiles int, totalSize int64, lastUpdate time.Time) {
	projects := c.getProjectStatus()
	
	for _, proj := range projects {
		totalFiles += proj.FilesIndexed
		totalSize += proj.Size
		if proj.LastIndexed.After(lastUpdate) {
			lastUpdate = proj.LastIndexed
		}
	}
	
	return
}

func isCodeFile(ext string) bool {
	codeExts := []string{
		".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".java", ".c", ".cpp", 
		".h", ".hpp", ".cs", ".rb", ".php", ".swift", ".kt", ".rs", ".scala",
		".sh", ".bash", ".zsh", ".fish", ".ps1", ".yml", ".yaml", ".json",
		".xml", ".html", ".css", ".scss", ".sass", ".sql", ".r", ".m", ".mm",
	}
	
	for _, codeExt := range codeExts {
		if strings.EqualFold(ext, codeExt) {
			return true
		}
	}
	
	return false
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	
	diff := time.Since(t)
	
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}