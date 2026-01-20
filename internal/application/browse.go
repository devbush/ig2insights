package application

import (
	"context"

	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

// BrowseService handles account browsing operations
type BrowseService struct {
	fetcher ports.AccountFetcher
}

// NewBrowseService creates a new browse service
func NewBrowseService(fetcher ports.AccountFetcher) *BrowseService {
	return &BrowseService{fetcher: fetcher}
}

// GetAccount retrieves account information
func (s *BrowseService) GetAccount(ctx context.Context, username string) (*domain.Account, error) {
	return s.fetcher.GetAccount(ctx, username)
}

// ListReels retrieves reels from an account
func (s *BrowseService) ListReels(ctx context.Context, username string, sort domain.SortOrder, limit int) ([]*domain.Reel, error) {
	return s.fetcher.ListReels(ctx, username, sort, limit)
}
