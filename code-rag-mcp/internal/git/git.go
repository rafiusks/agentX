package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitTracker tracks changes using git
type GitTracker struct {
	repoPath string
}

// FileStatus represents git file status
type FileStatus int

const (
	StatusUntracked FileStatus = iota
	StatusModified
	StatusAdded
	StatusDeleted
	StatusRenamed
	StatusCopied
)

// ChangedFile represents a file changed in git
type ChangedFile struct {
	Path      string
	Status    FileStatus
	OldPath   string // For renames
	Staged    bool
	Untracked bool
}

// NewGitTracker creates a new git tracker
func NewGitTracker(repoPath string) (*GitTracker, error) {
	// Check if path is a git repository
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("not a git repository: %s", repoPath)
	}

	return &GitTracker{repoPath: repoPath}, nil
}

// GetChangedFiles returns files changed since a commit or ref
func (gt *GitTracker) GetChangedFiles(since string) ([]ChangedFile, error) {
	var files []ChangedFile

	// Get committed changes
	if since != "" {
		committed, err := gt.getDiffFiles(since, "HEAD")
		if err != nil {
			return nil, err
		}
		files = append(files, committed...)
	}

	// Get uncommitted changes
	uncommitted, err := gt.getStatus()
	if err != nil {
		return nil, err
	}
	files = append(files, uncommitted...)

	return files, nil
}

// GetModifiedSince returns files modified since a timestamp
func (gt *GitTracker) GetModifiedSince(since time.Time) ([]ChangedFile, error) {
	// Use git log to find commits since timestamp
	cmd := exec.Command("git", "log", "--since="+since.Format("2006-01-02 15:04:05"), "--pretty=format:%H", "-n", "1")
	cmd.Dir = gt.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	commit := strings.TrimSpace(string(output))
	if commit == "" {
		// No commits since timestamp, return working directory changes
		return gt.getStatus()
	}

	// Get changes since that commit
	return gt.GetChangedFiles(commit + "^")
}

// getDiffFiles gets files changed between two commits/refs
func (gt *GitTracker) getDiffFiles(from, to string) ([]ChangedFile, error) {
	cmd := exec.Command("git", "diff", "--name-status", from, to)
	cmd.Dir = gt.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	return gt.parseDiffOutput(output), nil
}

// getStatus gets current working directory status
func (gt *GitTracker) getStatus() ([]ChangedFile, error) {
	cmd := exec.Command("git", "status", "--porcelain", "-uall")
	cmd.Dir = gt.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status failed: %w", err)
	}

	return gt.parseStatusOutput(output), nil
}

// parseDiffOutput parses git diff --name-status output
func (gt *GitTracker) parseDiffOutput(output []byte) []ChangedFile {
	var files []ChangedFile
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		path := parts[1]

		var file ChangedFile
		file.Path = filepath.Join(gt.repoPath, path)

		switch status[0] {
		case 'A':
			file.Status = StatusAdded
		case 'M':
			file.Status = StatusModified
		case 'D':
			file.Status = StatusDeleted
		case 'R':
			file.Status = StatusRenamed
			if len(parts) >= 3 {
				file.OldPath = filepath.Join(gt.repoPath, parts[1])
				file.Path = filepath.Join(gt.repoPath, parts[2])
			}
		case 'C':
			file.Status = StatusCopied
			if len(parts) >= 3 {
				file.OldPath = filepath.Join(gt.repoPath, parts[1])
				file.Path = filepath.Join(gt.repoPath, parts[2])
			}
		default:
			continue
		}

		// Only include code files
		if isCodeFile(file.Path) {
			files = append(files, file)
		}
	}

	return files
}

// parseStatusOutput parses git status --porcelain output
func (gt *GitTracker) parseStatusOutput(output []byte) []ChangedFile {
	var files []ChangedFile
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 4 {
			continue
		}

		// Parse status codes
		indexStatus := line[0]
		workStatus := line[1]
		path := strings.TrimSpace(line[3:])

		// Handle renames (format: "R  old -> new")
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			if len(parts) == 2 {
				file := ChangedFile{
					Path:    filepath.Join(gt.repoPath, parts[1]),
					OldPath: filepath.Join(gt.repoPath, parts[0]),
					Status:  StatusRenamed,
					Staged:  indexStatus != ' ' && indexStatus != '?',
				}
				if isCodeFile(file.Path) {
					files = append(files, file)
				}
				continue
			}
		}

		file := ChangedFile{
			Path: filepath.Join(gt.repoPath, path),
		}

		// Determine status
		if indexStatus == '?' && workStatus == '?' {
			file.Status = StatusUntracked
			file.Untracked = true
		} else if indexStatus == 'A' || workStatus == 'A' {
			file.Status = StatusAdded
			file.Staged = indexStatus == 'A'
		} else if indexStatus == 'M' || workStatus == 'M' {
			file.Status = StatusModified
			file.Staged = indexStatus == 'M'
		} else if indexStatus == 'D' || workStatus == 'D' {
			file.Status = StatusDeleted
			file.Staged = indexStatus == 'D'
		} else if indexStatus == 'R' {
			file.Status = StatusRenamed
			file.Staged = true
		} else {
			continue
		}

		// Only include code files
		if isCodeFile(file.Path) {
			files = append(files, file)
		}
	}

	return files
}

// GetCurrentBranch returns the current git branch
func (gt *GitTracker) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = gt.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetLastCommit returns the last commit SHA
func (gt *GitTracker) GetLastCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = gt.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get last commit: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetDiffContent gets the actual diff content for a file
func (gt *GitTracker) GetDiffContent(filePath string) (string, error) {
	relPath, _ := filepath.Rel(gt.repoPath, filePath)
	cmd := exec.Command("git", "diff", "HEAD", "--", relPath)
	cmd.Dir = gt.repoPath
	output, err := cmd.Output()
	if err != nil {
		// Try diff against index if working file
		cmd = exec.Command("git", "diff", "--", relPath)
		cmd.Dir = gt.repoPath
		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get diff: %w", err)
		}
	}
	return string(output), nil
}

// IsIgnored checks if a file is git-ignored
func (gt *GitTracker) IsIgnored(filePath string) bool {
	relPath, _ := filepath.Rel(gt.repoPath, filePath)
	cmd := exec.Command("git", "check-ignore", relPath)
	cmd.Dir = gt.repoPath
	return cmd.Run() == nil // Returns nil if file is ignored
}

// isCodeFile checks if a file is a code file based on extension
func isCodeFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	codeExts := []string{
		".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".java", ".c", ".cpp",
		".h", ".hpp", ".cs", ".rb", ".php", ".swift", ".kt", ".rs", ".scala",
		".sh", ".bash", ".zsh", ".yml", ".yaml", ".json", ".xml", ".html",
		".css", ".scss", ".sql", ".proto", ".graphql", ".vue", ".svelte",
	}

	for _, codeExt := range codeExts {
		if ext == codeExt {
			return true
		}
	}

	return false
}