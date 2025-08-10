package watcher

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
)

// RenameTracker tracks file renames using inode numbers
type RenameTracker struct {
	mu sync.RWMutex
	// Map from inode to file path
	inodeToPath map[uint64]string
	// Map from path to inode
	pathToInode map[string]uint64
	// Recent deletions that might be renames
	recentDeletes map[uint64]DeleteInfo
	// Rename detection window
	renameWindow time.Duration
}

// DeleteInfo tracks a recently deleted file
type DeleteInfo struct {
	Path      string
	Inode     uint64
	DeletedAt time.Time
	Size      int64
	ModTime   time.Time
}

// RenameEvent represents a detected rename
type RenameEvent struct {
	OldPath string
	NewPath string
	Inode   uint64
}

// NewRenameTracker creates a new rename tracker
func NewRenameTracker() *RenameTracker {
	rt := &RenameTracker{
		inodeToPath:   make(map[uint64]string),
		pathToInode:   make(map[string]uint64),
		recentDeletes: make(map[uint64]DeleteInfo),
		renameWindow:  500 * time.Millisecond, // Window to detect rename pairs
	}
	
	// Start cleanup goroutine for old delete records
	go rt.cleanupLoop()
	
	return rt
}

// TrackFile records a file's inode for rename detection
func (rt *RenameTracker) TrackFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	
	inode := getInode(info)
	if inode == 0 {
		return fmt.Errorf("failed to get inode for %s", path)
	}
	
	rt.mu.Lock()
	defer rt.mu.Unlock()
	
	// Check if this inode was previously tracked with a different path
	if oldPath, exists := rt.inodeToPath[inode]; exists && oldPath != path {
		// This is likely a rename or hard link
		delete(rt.pathToInode, oldPath)
	}
	
	// Update mappings
	rt.inodeToPath[inode] = path
	rt.pathToInode[path] = inode
	
	return nil
}

// HandleDelete processes a file deletion that might be part of a rename
func (rt *RenameTracker) HandleDelete(path string) *DeleteInfo {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	
	inode, exists := rt.pathToInode[path]
	if !exists {
		// Try to get inode from file system (might still exist briefly)
		if info, err := os.Stat(path); err == nil {
			inode = getInode(info)
		}
	}
	
	if inode == 0 {
		return nil
	}
	
	// Get file info before it's gone
	var size int64
	var modTime time.Time
	if info, err := os.Stat(path); err == nil {
		size = info.Size()
		modTime = info.ModTime()
	}
	
	deleteInfo := DeleteInfo{
		Path:      path,
		Inode:     inode,
		DeletedAt: time.Now(),
		Size:      size,
		ModTime:   modTime,
	}
	
	// Store in recent deletes
	rt.recentDeletes[inode] = deleteInfo
	
	// Clean up mappings
	delete(rt.pathToInode, path)
	// Don't delete from inodeToPath yet - might be a rename
	
	return &deleteInfo
}

// HandleCreate processes a file creation that might be part of a rename
func (rt *RenameTracker) HandleCreate(path string) *RenameEvent {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	
	inode := getInode(info)
	if inode == 0 {
		return nil
	}
	
	rt.mu.Lock()
	defer rt.mu.Unlock()
	
	// Check if this inode was recently deleted (potential rename)
	if deleteInfo, exists := rt.recentDeletes[inode]; exists {
		// Check if the deletion was recent enough
		if time.Since(deleteInfo.DeletedAt) <= rt.renameWindow {
			// This is likely a rename!
			delete(rt.recentDeletes, inode)
			
			// Update mappings
			rt.inodeToPath[inode] = path
			rt.pathToInode[path] = inode
			
			return &RenameEvent{
				OldPath: deleteInfo.Path,
				NewPath: path,
				Inode:   inode,
			}
		}
	}
	
	// Check if this inode was previously tracked (might be a hard link or moved from outside)
	if oldPath, exists := rt.inodeToPath[inode]; exists && oldPath != path {
		// Update mappings
		delete(rt.pathToInode, oldPath)
		rt.inodeToPath[inode] = path
		rt.pathToInode[path] = inode
		
		// This could be a rename from a path we weren't tracking
		return &RenameEvent{
			OldPath: oldPath,
			NewPath: path,
			Inode:   inode,
		}
	}
	
	// New file, track it
	rt.inodeToPath[inode] = path
	rt.pathToInode[path] = inode
	
	return nil
}

// DetectBulkRenames detects renames in a batch of changes
func (rt *RenameTracker) DetectBulkRenames(deletes []string, creates []string) []RenameEvent {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	
	var renames []RenameEvent
	
	// Build a map of deleted files by size and mod time
	deletesBySize := make(map[string][]*DeleteInfo)
	for _, path := range deletes {
		if info, err := os.Stat(path); err == nil {
			inode := getInode(info)
			key := fmt.Sprintf("%d_%d", info.Size(), info.ModTime().Unix())
			deletesBySize[key] = append(deletesBySize[key], &DeleteInfo{
				Path:    path,
				Inode:   inode,
				Size:    info.Size(),
				ModTime: info.ModTime(),
			})
		}
	}
	
	// Match creates with deletes
	for _, createPath := range creates {
		if info, err := os.Stat(createPath); err == nil {
			key := fmt.Sprintf("%d_%d", info.Size(), info.ModTime().Unix())
			if deletes, exists := deletesBySize[key]; exists && len(deletes) > 0 {
				// Potential rename match
				deleteInfo := deletes[0]
				deletesBySize[key] = deletes[1:]
				
				inode := getInode(info)
				renames = append(renames, RenameEvent{
					OldPath: deleteInfo.Path,
					NewPath: createPath,
					Inode:   inode,
				})
				
				// Update tracking
				rt.inodeToPath[inode] = createPath
				rt.pathToInode[createPath] = inode
				delete(rt.pathToInode, deleteInfo.Path)
			}
		}
	}
	
	return renames
}

// GetInode returns the inode for a path
func (rt *RenameTracker) GetInode(path string) (uint64, error) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	
	if inode, exists := rt.pathToInode[path]; exists {
		return inode, nil
	}
	
	// Try to get from file system
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	
	return getInode(info), nil
}

// GetPath returns the current path for an inode
func (rt *RenameTracker) GetPath(inode uint64) (string, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	
	path, exists := rt.inodeToPath[inode]
	return path, exists
}

// cleanupLoop periodically removes old delete records
func (rt *RenameTracker) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		rt.mu.Lock()
		now := time.Now()
		for inode, info := range rt.recentDeletes {
			if now.Sub(info.DeletedAt) > rt.renameWindow*2 {
				delete(rt.recentDeletes, inode)
				// Also clean up from inodeToPath if it still points to the old path
				if path, exists := rt.inodeToPath[inode]; exists && path == info.Path {
					delete(rt.inodeToPath, inode)
				}
			}
		}
		rt.mu.Unlock()
	}
}

// Clear removes all tracking information
func (rt *RenameTracker) Clear() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	
	rt.inodeToPath = make(map[uint64]string)
	rt.pathToInode = make(map[string]uint64)
	rt.recentDeletes = make(map[uint64]DeleteInfo)
}

// Stats returns tracking statistics
func (rt *RenameTracker) Stats() map[string]int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	
	return map[string]int{
		"tracked_files":  len(rt.pathToInode),
		"tracked_inodes": len(rt.inodeToPath),
		"recent_deletes": len(rt.recentDeletes),
	}
}

// Platform-specific inode extraction

// getInode extracts the inode number from FileInfo
func getInode(info os.FileInfo) uint64 {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return stat.Ino
	}
	return 0
}

// InodeInfo provides detailed inode information
type InodeInfo struct {
	Inode    uint64
	Device   uint64
	Mode     uint32
	Nlink    uint64
	UID      uint32
	GID      uint32
	Size     int64
	Atime    time.Time
	Mtime    time.Time
	Ctime    time.Time
}

// GetInodeInfo returns detailed inode information for a file
func GetInodeInfo(path string) (*InodeInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, fmt.Errorf("failed to get stat info for %s", path)
	}
	
	return &InodeInfo{
		Inode:  stat.Ino,
		Device: uint64(stat.Dev),
		Mode:   uint32(stat.Mode),
		Nlink:  uint64(stat.Nlink),
		UID:    stat.Uid,
		GID:    stat.Gid,
		Size:   stat.Size,
		Atime:  time.Unix(stat.Atimespec.Sec, stat.Atimespec.Nsec),
		Mtime:  time.Unix(stat.Mtimespec.Sec, stat.Mtimespec.Nsec),
		Ctime:  time.Unix(stat.Ctimespec.Sec, stat.Ctimespec.Nsec),
	}, nil
}

// AreHardLinked checks if two paths are hard links to the same inode
func AreHardLinked(path1, path2 string) (bool, error) {
	info1, err := GetInodeInfo(path1)
	if err != nil {
		return false, err
	}
	
	info2, err := GetInodeInfo(path2)
	if err != nil {
		return false, err
	}
	
	return info1.Inode == info2.Inode && info1.Device == info2.Device, nil
}