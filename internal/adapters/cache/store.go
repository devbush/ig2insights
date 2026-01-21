package cache

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

const (
	dirPerm  = 0755
	filePerm = 0644
	metaName = "meta.json"
)

// FileCache implements ports.CacheStore using the local filesystem.
type FileCache struct {
	baseDir string
}

// NewFileCache creates a new file-based cache store.
func NewFileCache(baseDir string) *FileCache {
	return &FileCache{baseDir: baseDir}
}

// cacheEntry is the on-disk representation of a cached item.
type cacheEntry struct {
	Reel          *domain.Reel       `json:"reel"`
	Transcript    *domain.Transcript `json:"transcript"`
	AudioPath     string             `json:"audio_path"`
	VideoPath     string             `json:"video_path"`
	ThumbnailPath string             `json:"thumbnail_path"`
	CreatedAt     time.Time          `json:"created_at"`
	ExpiresAt     time.Time          `json:"expires_at"`
}

func (c *FileCache) GetCacheDir(reelID string) string {
	return filepath.Join(c.baseDir, reelID)
}

func (c *FileCache) metaPath(reelID string) string {
	return filepath.Join(c.GetCacheDir(reelID), metaName)
}

func (c *FileCache) Get(ctx context.Context, reelID string) (*ports.CachedItem, error) {
	data, err := os.ReadFile(c.metaPath(reelID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.ErrCacheMiss
		}
		return nil, err
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil, domain.ErrCacheExpired
	}

	return &ports.CachedItem{
		Reel:          entry.Reel,
		Transcript:    entry.Transcript,
		AudioPath:     entry.AudioPath,
		VideoPath:     entry.VideoPath,
		ThumbnailPath: entry.ThumbnailPath,
		CreatedAt:     entry.CreatedAt,
		ExpiresAt:     entry.ExpiresAt,
	}, nil
}

func (c *FileCache) Set(ctx context.Context, reelID string, item *ports.CachedItem) error {
	if err := os.MkdirAll(c.GetCacheDir(reelID), dirPerm); err != nil {
		return err
	}

	entry := cacheEntry{
		Reel:          item.Reel,
		Transcript:    item.Transcript,
		AudioPath:     item.AudioPath,
		VideoPath:     item.VideoPath,
		ThumbnailPath: item.ThumbnailPath,
		CreatedAt:     item.CreatedAt,
		ExpiresAt:     item.ExpiresAt,
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.metaPath(reelID), data, filePerm)
}

func (c *FileCache) Delete(ctx context.Context, reelID string) error {
	return os.RemoveAll(c.GetCacheDir(reelID))
}

func (c *FileCache) CleanExpired(ctx context.Context) (int, error) {
	entries, err := c.readCacheDirs()
	if err != nil {
		return 0, err
	}

	cleaned := 0
	for _, entry := range entries {
		reelID := entry.Name()
		_, err := c.Get(ctx, reelID)
		if errors.Is(err, domain.ErrCacheExpired) {
			if deleteErr := c.Delete(ctx, reelID); deleteErr == nil {
				cleaned++
			}
		}
	}

	return cleaned, nil
}

func (c *FileCache) Clear(ctx context.Context) error {
	entries, err := c.readCacheDirs()
	if err != nil {
		return err
	}

	for _, entry := range entries {
		_ = os.RemoveAll(filepath.Join(c.baseDir, entry.Name()))
	}

	return nil
}

func (c *FileCache) Stats(ctx context.Context) (itemCount int, totalSize int64, err error) {
	entries, err := c.readCacheDirs()
	if err != nil {
		return 0, 0, err
	}

	for _, entry := range entries {
		itemCount++
		totalSize += c.dirSize(filepath.Join(c.baseDir, entry.Name()))
	}

	return itemCount, totalSize, nil
}

// readCacheDirs returns all directory entries in the cache base directory.
// Returns an empty slice if the directory does not exist.
func (c *FileCache) readCacheDirs() ([]os.DirEntry, error) {
	entries, err := os.ReadDir(c.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	dirs := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry)
		}
	}
	return dirs, nil
}

// dirSize calculates the total size of all files in a directory.
func (c *FileCache) dirSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

var _ ports.CacheStore = (*FileCache)(nil)
