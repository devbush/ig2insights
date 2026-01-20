package ytdlp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/devbush/ig2insights/internal/config"
	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

// Downloader implements VideoDownloader and AccountFetcher using yt-dlp
type Downloader struct {
	binPath string
}

// NewDownloader creates a new yt-dlp downloader
func NewDownloader() *Downloader {
	return &Downloader{}
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "yt-dlp.exe"
	}
	return "yt-dlp"
}

func buildReelURL(reelID string) string {
	return fmt.Sprintf("https://www.instagram.com/p/%s/", reelID)
}

func (d *Downloader) findBinary() string {
	// Check bundled location first
	bundled := filepath.Join(config.BinDir(), binaryName())
	if _, err := os.Stat(bundled); err == nil {
		return bundled
	}

	// Check system PATH
	if path, err := exec.LookPath(binaryName()); err == nil {
		return path
	}

	return ""
}

func (d *Downloader) GetBinaryPath() string {
	if d.binPath != "" {
		return d.binPath
	}
	d.binPath = d.findBinary()
	return d.binPath
}

func (d *Downloader) IsAvailable() bool {
	return d.GetBinaryPath() != ""
}

func (d *Downloader) Download(ctx context.Context, reelID string, destDir string) (*ports.DownloadResult, error) {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return nil, fmt.Errorf("yt-dlp not found")
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	url := buildReelURL(reelID)
	outputTemplate := filepath.Join(destDir, "video.%(ext)s")

	// Run yt-dlp with JSON output for metadata
	args := []string{
		"--no-warnings",
		"--print-json",
		"-o", outputTemplate,
		url,
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "Private video") || strings.Contains(stderr, "Video unavailable") {
				return nil, domain.ErrReelNotFound
			}
			if strings.Contains(stderr, "rate") || strings.Contains(stderr, "429") {
				return nil, domain.ErrRateLimited
			}
			return nil, fmt.Errorf("yt-dlp failed: %s", stderr)
		}
		return nil, fmt.Errorf("yt-dlp failed: %w", err)
	}

	// Parse JSON output for metadata
	var info struct {
		ID                 string  `json:"id"`
		Title              string  `json:"title"`
		Uploader           string  `json:"uploader"`
		Duration           float64 `json:"duration"`
		ViewCount          int64   `json:"view_count"`
		Ext                string  `json:"ext"`
		RequestedDownloads []struct {
			Filepath string `json:"filepath"`
		} `json:"requested_downloads"`
	}

	if err := json.Unmarshal(output, &info); err != nil {
		// Try to find the video file anyway
		matches, _ := filepath.Glob(filepath.Join(destDir, "video.*"))
		if len(matches) > 0 {
			return &ports.DownloadResult{
				VideoPath: matches[0],
				Reel: &domain.Reel{
					ID:        reelID,
					FetchedAt: time.Now(),
				},
			}, nil
		}
		return nil, fmt.Errorf("failed to parse yt-dlp output: %w", err)
	}

	videoPath := filepath.Join(destDir, fmt.Sprintf("video.%s", info.Ext))
	if len(info.RequestedDownloads) > 0 {
		videoPath = info.RequestedDownloads[0].Filepath
	}

	return &ports.DownloadResult{
		VideoPath: videoPath,
		Reel: &domain.Reel{
			ID:              reelID,
			URL:             url,
			Author:          info.Uploader,
			Title:           info.Title,
			DurationSeconds: int(info.Duration),
			ViewCount:       info.ViewCount,
			FetchedAt:       time.Now(),
		},
	}, nil
}

func (d *Downloader) Install(ctx context.Context, progress func(downloaded, total int64)) error {
	binDir := config.BinDir()
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	downloadURL := d.getDownloadURL()
	destPath := filepath.Join(binDir, binaryName())

	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download yt-dlp: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download yt-dlp: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	var downloaded int64

	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
			if progress != nil {
				progress(downloaded, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	// Make executable on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(destPath, 0755); err != nil {
			return err
		}
	}

	d.binPath = destPath
	return nil
}

func (d *Downloader) getDownloadURL() string {
	base := "https://github.com/yt-dlp/yt-dlp/releases/latest/download/"

	switch runtime.GOOS {
	case "windows":
		return base + "yt-dlp.exe"
	case "darwin":
		return base + "yt-dlp_macos"
	default:
		return base + "yt-dlp"
	}
}

func (d *Downloader) Update(ctx context.Context) error {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return fmt.Errorf("yt-dlp not installed")
	}

	cmd := exec.CommandContext(ctx, binPath, "-U")
	return cmd.Run()
}

// Ensure Downloader implements interfaces
var _ ports.VideoDownloader = (*Downloader)(nil)
