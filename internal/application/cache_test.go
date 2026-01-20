package application

import (
	"context"
	"errors"
	"testing"

	"github.com/devbush/ig2insights/internal/ports"
)

// mockCacheStore implements ports.CacheStore for cache service testing
type mockCacheStore struct {
	itemCount    int
	totalSize    int64
	cleanedCount int
	statsErr     error
	cleanErr     error
	clearErr     error
}

func (m *mockCacheStore) Get(ctx context.Context, reelID string) (*ports.CachedItem, error) {
	return nil, nil
}

func (m *mockCacheStore) Set(ctx context.Context, reelID string, item *ports.CachedItem) error {
	return nil
}

func (m *mockCacheStore) Delete(ctx context.Context, reelID string) error {
	return nil
}

func (m *mockCacheStore) CleanExpired(ctx context.Context) (int, error) {
	if m.cleanErr != nil {
		return 0, m.cleanErr
	}
	return m.cleanedCount, nil
}

func (m *mockCacheStore) Clear(ctx context.Context) error {
	return m.clearErr
}

func (m *mockCacheStore) GetCacheDir(reelID string) string {
	return "/tmp/cache/" + reelID
}

func (m *mockCacheStore) Stats(ctx context.Context) (int, int64, error) {
	if m.statsErr != nil {
		return 0, 0, m.statsErr
	}
	return m.itemCount, m.totalSize, nil
}

func TestCacheService_Stats(t *testing.T) {
	cache := &mockCacheStore{
		itemCount: 5,
		totalSize: 1024 * 1024 * 10, // 10MB
	}
	svc := NewCacheService(cache)

	ctx := context.Background()
	stats, err := svc.Stats(ctx)

	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	if stats.ItemCount != 5 {
		t.Errorf("ItemCount = %d, want 5", stats.ItemCount)
	}

	if stats.TotalSize != 1024*1024*10 {
		t.Errorf("TotalSize = %d, want %d", stats.TotalSize, 1024*1024*10)
	}
}

func TestCacheService_Stats_Error(t *testing.T) {
	expectedErr := errors.New("failed to get stats")
	cache := &mockCacheStore{statsErr: expectedErr}
	svc := NewCacheService(cache)

	ctx := context.Background()
	_, err := svc.Stats(ctx)

	if err == nil {
		t.Fatal("Stats() expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("Stats() error = %v, want %v", err, expectedErr)
	}
}

func TestCacheService_CleanExpired(t *testing.T) {
	cache := &mockCacheStore{cleanedCount: 3}
	svc := NewCacheService(cache)

	ctx := context.Background()
	count, err := svc.CleanExpired(ctx)

	if err != nil {
		t.Fatalf("CleanExpired() error = %v", err)
	}

	if count != 3 {
		t.Errorf("CleanExpired() = %d, want 3", count)
	}
}

func TestCacheService_CleanExpired_Error(t *testing.T) {
	expectedErr := errors.New("failed to clean")
	cache := &mockCacheStore{cleanErr: expectedErr}
	svc := NewCacheService(cache)

	ctx := context.Background()
	_, err := svc.CleanExpired(ctx)

	if err == nil {
		t.Fatal("CleanExpired() expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("CleanExpired() error = %v, want %v", err, expectedErr)
	}
}

func TestCacheService_Clear(t *testing.T) {
	cache := &mockCacheStore{}
	svc := NewCacheService(cache)

	ctx := context.Background()
	err := svc.Clear(ctx)

	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
}

func TestCacheService_Clear_Error(t *testing.T) {
	expectedErr := errors.New("failed to clear")
	cache := &mockCacheStore{clearErr: expectedErr}
	svc := NewCacheService(cache)

	ctx := context.Background()
	err := svc.Clear(ctx)

	if err == nil {
		t.Fatal("Clear() expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("Clear() error = %v, want %v", err, expectedErr)
	}
}
