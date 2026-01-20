package cache

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

type FileCache struct {
	baseDir string
	ttl     time.Duration
}

func NewFileCache(baseDir string, ttl time.Duration) *FileCache {
	return &FileCache{
		baseDir: baseDir,
		ttl:     ttl,
	}
}

type metaFile struct {
	Reel       *domain.Reel       `json:"reel"`
	Transcript *domain.Transcript `json:"transcript"`
	VideoPath  string             `json:"video_path"`
	CreatedAt  time.Time          `json:"created_at"`
	ExpiresAt  time.Time          `json:"expires_at"`
}

func (c *FileCache) GetCacheDir(reelID string) string {
	return filepath.Join(c.baseDir, reelID)
}

func (c *FileCache) metaPath(reelID string) string {
	return filepath.Join(c.GetCacheDir(reelID), "meta.json")
}

func (c *FileCache) Get(ctx context.Context, reelID string) (*ports.CachedItem, error) {
	metaPath := c.metaPath(reelID)

	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.ErrCacheMiss
		}
		return nil, err
	}

	var meta metaFile
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	if time.Now().After(meta.ExpiresAt) {
		return nil, domain.ErrCacheExpired
	}

	return &ports.CachedItem{
		Reel:       meta.Reel,
		Transcript: meta.Transcript,
		VideoPath:  meta.VideoPath,
		CreatedAt:  meta.CreatedAt,
		ExpiresAt:  meta.ExpiresAt,
	}, nil
}

func (c *FileCache) Set(ctx context.Context, reelID string, item *ports.CachedItem) error {
	cacheDir := c.GetCacheDir(reelID)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	meta := metaFile{
		Reel:       item.Reel,
		Transcript: item.Transcript,
		VideoPath:  item.VideoPath,
		CreatedAt:  item.CreatedAt,
		ExpiresAt:  item.ExpiresAt,
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.metaPath(reelID), data, 0644)
}

func (c *FileCache) Delete(ctx context.Context, reelID string) error {
	return os.RemoveAll(c.GetCacheDir(reelID))
}

func (c *FileCache) CleanExpired(ctx context.Context) (int, error) {
	entries, err := os.ReadDir(c.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	cleaned := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		reelID := entry.Name()
		_, err := c.Get(ctx, reelID)
		if err == domain.ErrCacheExpired {
			if err := c.Delete(ctx, reelID); err == nil {
				cleaned++
			}
		}
	}

	return cleaned, nil
}

func (c *FileCache) Clear(ctx context.Context) error {
	entries, err := os.ReadDir(c.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			_ = os.RemoveAll(filepath.Join(c.baseDir, entry.Name()))
		}
	}

	return nil
}

func (c *FileCache) Stats(ctx context.Context) (itemCount int, totalSize int64, err error) {
	entries, err := os.ReadDir(c.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		itemCount++

		dirPath := filepath.Join(c.baseDir, entry.Name())
		_ = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				totalSize += info.Size()
			}
			return nil
		})
	}

	return itemCount, totalSize, nil
}

var _ ports.CacheStore = (*FileCache)(nil)
