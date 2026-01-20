package ports

import (
	"context"

	"github.com/devbush/ig2insights/internal/domain"
)

// DownloadResult contains the downloaded audio info
type DownloadResult struct {
	AudioPath string       // WAV audio file for transcription
	Reel      *domain.Reel // Populated with metadata from download
}

// VideoDownloader handles video download from Instagram
type VideoDownloader interface {
	// DownloadAudio extracts audio from a reel, returns path to WAV file
	DownloadAudio(ctx context.Context, reelID string, destDir string) (*DownloadResult, error)

	// DownloadVideo downloads the full video file (MP4 with audio)
	DownloadVideo(ctx context.Context, reelID string, destPath string) error

	// DownloadThumbnail downloads the video thumbnail
	DownloadThumbnail(ctx context.Context, reelID string, destPath string) error

	// IsAvailable checks if the downloader is ready (yt-dlp installed)
	IsAvailable() bool

	// GetBinaryPath returns path to yt-dlp binary
	GetBinaryPath() string

	// Install downloads and installs yt-dlp
	Install(ctx context.Context, progress func(downloaded, total int64)) error

	// Update updates yt-dlp to latest version
	Update(ctx context.Context) error

	// IsFFmpegAvailable checks if ffmpeg is installed
	IsFFmpegAvailable() bool

	// GetFFmpegPath returns path to ffmpeg binary
	GetFFmpegPath() string

	// InstallFFmpeg downloads and installs ffmpeg (Windows only)
	InstallFFmpeg(ctx context.Context, progress func(downloaded, total int64)) error

	// FFmpegInstructions returns platform-specific install instructions
	FFmpegInstructions() string
}
