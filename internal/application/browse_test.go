package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
)

// mockAccountFetcher implements ports.AccountFetcher for testing
type mockAccountFetcher struct {
	account *domain.Account
	reels   []*domain.Reel
	err     error
}

func (m *mockAccountFetcher) GetAccount(ctx context.Context, username string) (*domain.Account, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.account != nil {
		return m.account, nil
	}
	return &domain.Account{
		Username:  username,
		ReelCount: 10,
	}, nil
}

func (m *mockAccountFetcher) ListReels(ctx context.Context, username string, sort domain.SortOrder, limit int) ([]*domain.Reel, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.reels != nil {
		if limit > 0 && limit < len(m.reels) {
			return m.reels[:limit], nil
		}
		return m.reels, nil
	}
	return []*domain.Reel{
		{ID: "reel1", Author: username, Title: "Test Reel 1", FetchedAt: time.Now()},
		{ID: "reel2", Author: username, Title: "Test Reel 2", FetchedAt: time.Now()},
	}, nil
}

func TestBrowseService_GetAccount(t *testing.T) {
	fetcher := &mockAccountFetcher{}
	svc := NewBrowseService(fetcher)

	ctx := context.Background()
	account, err := svc.GetAccount(ctx, "testuser")

	if err != nil {
		t.Fatalf("GetAccount() error = %v", err)
	}

	if account.Username != "testuser" {
		t.Errorf("Username = %s, want 'testuser'", account.Username)
	}

	if account.ReelCount != 10 {
		t.Errorf("ReelCount = %d, want 10", account.ReelCount)
	}
}

func TestBrowseService_GetAccount_Error(t *testing.T) {
	expectedErr := errors.New("account not found")
	fetcher := &mockAccountFetcher{err: expectedErr}
	svc := NewBrowseService(fetcher)

	ctx := context.Background()
	_, err := svc.GetAccount(ctx, "nonexistent")

	if err == nil {
		t.Fatal("GetAccount() expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("GetAccount() error = %v, want %v", err, expectedErr)
	}
}

func TestBrowseService_ListReels(t *testing.T) {
	fetcher := &mockAccountFetcher{}
	svc := NewBrowseService(fetcher)

	ctx := context.Background()
	reels, err := svc.ListReels(ctx, "testuser", domain.SortLatest, 10)

	if err != nil {
		t.Fatalf("ListReels() error = %v", err)
	}

	if len(reels) != 2 {
		t.Errorf("ListReels() returned %d reels, want 2", len(reels))
	}

	if reels[0].ID != "reel1" {
		t.Errorf("First reel ID = %s, want 'reel1'", reels[0].ID)
	}
}

func TestBrowseService_ListReels_WithLimit(t *testing.T) {
	fetcher := &mockAccountFetcher{
		reels: []*domain.Reel{
			{ID: "reel1", Author: "testuser", Title: "Reel 1"},
			{ID: "reel2", Author: "testuser", Title: "Reel 2"},
			{ID: "reel3", Author: "testuser", Title: "Reel 3"},
		},
	}
	svc := NewBrowseService(fetcher)

	ctx := context.Background()
	reels, err := svc.ListReels(ctx, "testuser", domain.SortMostViewed, 2)

	if err != nil {
		t.Fatalf("ListReels() error = %v", err)
	}

	if len(reels) != 2 {
		t.Errorf("ListReels() with limit 2 returned %d reels, want 2", len(reels))
	}
}

func TestBrowseService_ListReels_Error(t *testing.T) {
	expectedErr := errors.New("failed to fetch reels")
	fetcher := &mockAccountFetcher{err: expectedErr}
	svc := NewBrowseService(fetcher)

	ctx := context.Background()
	_, err := svc.ListReels(ctx, "testuser", domain.SortLatest, 10)

	if err == nil {
		t.Fatal("ListReels() expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("ListReels() error = %v, want %v", err, expectedErr)
	}
}
