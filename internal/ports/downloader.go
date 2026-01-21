package ports

import (
	"context"

	"github.com/devbush/ig2insights/internal/domain"
)

// DownloadResult contains the result of an audio download operation.
type DownloadResult struct {
	AudioPath string       // path to the WAV audio file
	Reel      *domain.Reel // reel metadata populated from download
}

// VideoDownloader handles video download from Instagram.
type VideoDownloader interface {
	// Download operations

	// DownloadAudio extracts audio from a reel and returns the path to the WAV file.
	DownloadAudio(ctx context.Context, reelID string, destDir string) (*DownloadResult, error)

	// DownloadVideo downloads the full video file (MP4 with audio).
	DownloadVideo(ctx context.Context, reelID string, destPath string) error

	// DownloadThumbnail downloads the video thumbnail image.
	DownloadThumbnail(ctx context.Context, reelID string, destPath string) error

	// yt-dlp management

	// IsAvailable checks if yt-dlp is installed and ready.
	IsAvailable() bool

	// GetBinaryPath returns the path to the yt-dlp binary.
	GetBinaryPath() string

	// Install downloads and installs yt-dlp, reporting progress via callback.
	Install(ctx context.Context, progress func(downloaded, total int64)) error

	// Update updates yt-dlp to the latest version.
	Update(ctx context.Context) error

	// ffmpeg management

	// IsFFmpegAvailable checks if ffmpeg is installed.
	IsFFmpegAvailable() bool

	// GetFFmpegPath returns the path to the ffmpeg binary.
	GetFFmpegPath() string

	// InstallFFmpeg downloads and installs ffmpeg (Windows only).
	InstallFFmpeg(ctx context.Context, progress func(downloaded, total int64)) error

	// FFmpegInstructions returns platform-specific installation instructions.
	FFmpegInstructions() string
}
