package watcher

import (
	"encoding/json"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

// HashStore persists file hashes using BoltDB
type HashStore struct {
	db *bolt.DB
}

// FileHashEntry stores hash and metadata for a file
type FileHashEntry struct {
	Hash         string    `json:"hash"`
	LastModified time.Time `json:"last_modified"`
	Size         int64     `json:"size"`
}

const (
	bucketHashes = "file_hashes"
	bucketMeta   = "metadata"
)

// NewHashStore creates a new hash store
func NewHashStore(dbPath string) (*HashStore, error) {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open hash store: %w", err)
	}

	// Create buckets if they don't exist
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketHashes)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketMeta)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create buckets: %w", err)
	}

	return &HashStore{db: db}, nil
}

// Close closes the database
func (hs *HashStore) Close() error {
	return hs.db.Close()
}

// GetHash retrieves the hash for a file
func (hs *HashStore) GetHash(filePath string) (string, error) {
	var hash string
	err := hs.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketHashes))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		data := bucket.Get([]byte(filePath))
		if data == nil {
			return nil // Not found
		}

		var entry FileHashEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return err
		}
		hash = entry.Hash
		return nil
	})
	return hash, err
}

// SetHash stores the hash for a file
func (hs *HashStore) SetHash(filePath string, hash string, modTime time.Time, size int64) error {
	return hs.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketHashes))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		entry := FileHashEntry{
			Hash:         hash,
			LastModified: modTime,
			Size:         size,
		}

		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(filePath), data)
	})
}

// DeleteHash removes the hash for a file
func (hs *HashStore) DeleteHash(filePath string) error {
	return hs.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketHashes))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}
		return bucket.Delete([]byte(filePath))
	})
}

// GetAllHashes retrieves all stored hashes
func (hs *HashStore) GetAllHashes() (map[string]string, error) {
	hashes := make(map[string]string)
	err := hs.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketHashes))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var entry FileHashEntry
			if err := json.Unmarshal(v, &entry); err != nil {
				return err
			}
			hashes[string(k)] = entry.Hash
			return nil
		})
	})
	return hashes, err
}

// SetLastIndexTime stores the last index time
func (hs *HashStore) SetLastIndexTime(collection string, t time.Time) error {
	return hs.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketMeta))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		data, err := t.MarshalBinary()
		if err != nil {
			return err
		}

		key := fmt.Sprintf("last_index_%s", collection)
		return bucket.Put([]byte(key), data)
	})
}

// GetLastIndexTime retrieves the last index time
func (hs *HashStore) GetLastIndexTime(collection string) (time.Time, error) {
	var t time.Time
	err := hs.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketMeta))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		key := fmt.Sprintf("last_index_%s", collection)
		data := bucket.Get([]byte(key))
		if data == nil {
			return nil // Not found
		}

		return t.UnmarshalBinary(data)
	})
	return t, err
}

// GetChangedFiles returns files that have changed since last check
func (hs *HashStore) GetChangedFiles(currentHashes map[string]string) ([]string, []string, []string) {
	var created, modified, deleted []string

	storedHashes, _ := hs.GetAllHashes()

	// Check for new and modified files
	for path, hash := range currentHashes {
		if storedHash, exists := storedHashes[path]; !exists {
			created = append(created, path)
		} else if storedHash != hash {
			modified = append(modified, path)
		}
	}

	// Check for deleted files
	for path := range storedHashes {
		if _, exists := currentHashes[path]; !exists {
			deleted = append(deleted, path)
		}
	}

	return created, modified, deleted
}