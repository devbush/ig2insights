package ports

import (
	"context"

	"github.com/devbush/ig2insights/internal/domain"
)

// DownloadResult contains the downloaded video info
type DownloadResult struct {
	VideoPath string
	Reel      *domain.Reel // Populated with metadata from download
}

// VideoDownloader handles video download from Instagram
type VideoDownloader interface {
	// Download fetches a video by reel ID, returns path to downloaded file
	Download(ctx context.Context, reelID string, destDir string) (*DownloadResult, error)

	// IsAvailable checks if the downloader is ready (yt-dlp installed)
	IsAvailable() bool

	// GetBinaryPath returns path to yt-dlp binary
	GetBinaryPath() string

	// Install downloads and installs yt-dlp
	Install(ctx context.Context, progress func(downloaded, total int64)) error

	// Update updates yt-dlp to latest version
	Update(ctx context.Context) error
}
