package ports

import (
	"context"

	"github.com/devbush/ig2insights/internal/domain"
)

// AccountFetcher retrieves Instagram account information
type AccountFetcher interface {
	// GetAccount fetches account info including reel count
	GetAccount(ctx context.Context, username string) (*domain.Account, error)

	// ListReels fetches reels from an account
	ListReels(ctx context.Context, username string, sort domain.SortOrder, limit int) ([]*domain.Reel, error)
}
