package application

import (
	"context"

	"github.com/devbush/ig2insights/internal/ports"
)

// CacheStats holds cache statistics
type CacheStats struct {
	ItemCount int
	TotalSize int64
}

// CacheService handles cache management operations
type CacheService struct {
	cache ports.CacheStore
}

// NewCacheService creates a new cache service
func NewCacheService(cache ports.CacheStore) *CacheService {
	return &CacheService{cache: cache}
}

// Stats returns cache statistics
func (s *CacheService) Stats(ctx context.Context) (*CacheStats, error) {
	count, size, err := s.cache.Stats(ctx)
	if err != nil {
		return nil, err
	}
	return &CacheStats{
		ItemCount: count,
		TotalSize: size,
	}, nil
}

// CleanExpired removes expired cache entries
func (s *CacheService) CleanExpired(ctx context.Context) (int, error) {
	return s.cache.CleanExpired(ctx)
}

// Clear removes all cache entries
func (s *CacheService) Clear(ctx context.Context) error {
	return s.cache.Clear(ctx)
}
