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
	"sort"
	"strings"
	"time"

	"github.com/bodgit/sevenzip"
	"github.com/devbush/ig2insights/internal/config"
	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

// Downloader implements VideoDownloader and AccountFetcher using yt-dlp
type Downloader struct {
	binPath    string
	ffmpegPath string
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

func ffmpegBinaryName() string {
	if runtime.GOOS == "windows" {
		return "ffmpeg.exe"
	}
	return "ffmpeg"
}

func ffprobeBinaryName() string {
	if runtime.GOOS == "windows" {
		return "ffprobe.exe"
	}
	return "ffprobe"
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

func (d *Downloader) findFFmpeg() string {
	// Check system PATH first (user may have ffmpeg installed)
	if path, err := exec.LookPath(ffmpegBinaryName()); err == nil {
		return path
	}

	// Check bundled location
	bundled := filepath.Join(config.BinDir(), ffmpegBinaryName())
	if _, err := os.Stat(bundled); err == nil {
		return bundled
	}

	return ""
}

func (d *Downloader) GetFFmpegPath() string {
	if d.ffmpegPath != "" {
		return d.ffmpegPath
	}
	d.ffmpegPath = d.findFFmpeg()
	return d.ffmpegPath
}

func (d *Downloader) IsFFmpegAvailable() bool {
	return d.GetFFmpegPath() != ""
}

func (d *Downloader) getFFmpegDownloadURL() string {
	if runtime.GOOS == "windows" {
		return "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.7z"
	}
	return ""
}

func (d *Downloader) FFmpegInstructions() string {
	switch runtime.GOOS {
	case "windows":
		return "" // Auto-download available
	case "darwin":
		return "ffmpeg not found. Install with:\n  brew install ffmpeg"
	default:
		return "ffmpeg not found. Install with:\n  sudo apt install ffmpeg  # Debian/Ubuntu\n  sudo dnf install ffmpeg  # Fedora"
	}
}

func (d *Downloader) Download(ctx context.Context, reelID string, destDir string) (*ports.DownloadResult, error) {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return nil, fmt.Errorf("yt-dlp not found")
	}

	// Check for ffmpeg (needed for audio extraction)
	if !d.IsFFmpegAvailable() {
		return nil, domain.ErrFFmpegNotFound
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	url := buildReelURL(reelID)
	outputTemplate := filepath.Join(destDir, "audio.%(ext)s")

	// Run yt-dlp with JSON output for metadata
	args := []string{
		"--no-warnings",
		"--print-json",
		"-x",                    // Extract audio
		"--audio-format", "wav", // Convert to wav (whisper-compatible)
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
		// Try to find the audio file anyway
		matches, _ := filepath.Glob(filepath.Join(destDir, "audio.*"))
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

	// Find the actual audio file - prefer WAV (requested format) over original
	audioPath := filepath.Join(destDir, "audio.wav")
	if _, err := os.Stat(audioPath); err != nil {
		// WAV not found, try RequestedDownloads path
		if len(info.RequestedDownloads) > 0 {
			audioPath = info.RequestedDownloads[0].Filepath
		} else {
			// Fall back to glob
			matches, _ := filepath.Glob(filepath.Join(destDir, "audio.*"))
			if len(matches) > 0 {
				audioPath = matches[0]
			}
		}
	}

	return &ports.DownloadResult{
		VideoPath: audioPath,
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

	// Use context-aware HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
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

	// Track success to clean up partial downloads on failure
	success := false
	defer func() {
		out.Close()
		if !success {
			os.Remove(destPath)
		}
	}()

	total := resp.ContentLength
	var downloaded int64

	buf := make([]byte, 32*1024)
	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

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

	success = true
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

func (d *Downloader) GetAccount(ctx context.Context, username string) (*domain.Account, error) {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return nil, fmt.Errorf("yt-dlp not found")
	}

	// Use playlist extraction to get account info
	url := fmt.Sprintf("https://www.instagram.com/%s/reels/", username)

	args := []string{
		"--no-warnings",
		"--flat-playlist",
		"--print-json",
		"-I", "1:1", // Only fetch first item to get playlist info
		url,
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "not found") || strings.Contains(stderr, "404") {
				return nil, domain.ErrAccountNotFound
			}
		}
		return nil, fmt.Errorf("failed to fetch account: %w", err)
	}

	// Count entries to estimate reel count
	// Note: ReelCount will be 0 or 1 due to the -I 1:1 flag limiting to first item.
	// This is a limitation - we only fetch one item to quickly verify the account exists.
	// A full reel count would require fetching the entire playlist which is slow.
	trimmed := strings.TrimSpace(string(output))
	var reelCount int
	if trimmed != "" {
		reelCount = len(strings.Split(trimmed, "\n"))
	}

	return &domain.Account{
		Username:  username,
		ReelCount: reelCount,
	}, nil
}

func (d *Downloader) ListReels(ctx context.Context, username string, sort domain.SortOrder, limit int) ([]*domain.Reel, error) {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return nil, fmt.Errorf("yt-dlp not found")
	}

	url := fmt.Sprintf("https://www.instagram.com/%s/reels/", username)

	args := []string{
		"--no-warnings",
		"--flat-playlist",
		"--print-json",
		"-I", fmt.Sprintf("1:%d", limit),
		url,
	}

	// Note: yt-dlp returns chronological order by default
	// For "most viewed", we'd need to fetch all and sort client-side
	// This is a limitation - Instagram doesn't expose a "top" endpoint easily

	cmd := exec.CommandContext(ctx, binPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "not found") || strings.Contains(stderr, "404") {
				return nil, domain.ErrAccountNotFound
			}
			if strings.Contains(stderr, "rate") || strings.Contains(stderr, "429") {
				return nil, domain.ErrRateLimited
			}
		}
		return nil, fmt.Errorf("failed to list reels: %w", err)
	}

	var reels []*domain.Reel
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		var info struct {
			ID        string  `json:"id"`
			Title     string  `json:"title"`
			Uploader  string  `json:"uploader"`
			Duration  float64 `json:"duration"`
			ViewCount int64   `json:"view_count"`
		}

		if err := json.Unmarshal([]byte(line), &info); err != nil {
			continue
		}

		reels = append(reels, &domain.Reel{
			ID:              info.ID,
			Author:          info.Uploader,
			Title:           info.Title,
			DurationSeconds: int(info.Duration),
			ViewCount:       info.ViewCount,
			FetchedAt:       time.Now(),
		})
	}

	// Sort by view count if requested
	if sort == domain.SortMostViewed {
		sortByViews(reels)
	}

	return reels, nil
}

func sortByViews(reels []*domain.Reel) {
	sort.Slice(reels, func(i, j int) bool {
		return reels[i].ViewCount > reels[j].ViewCount
	})
}

func (d *Downloader) InstallFFmpeg(ctx context.Context, progress func(downloaded, total int64)) error {
	downloadURL := d.getFFmpegDownloadURL()
	if downloadURL == "" {
		return fmt.Errorf("no prebuilt ffmpeg binary for %s.\n%s", runtime.GOOS, d.FFmpegInstructions())
	}

	binDir := config.BinDir()
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	// Download 7z to temp file
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download ffmpeg: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download ffmpeg: HTTP %d", resp.StatusCode)
	}

	// Download to temp file
	tmpFile, err := os.CreateTemp("", "ffmpeg-*.7z")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	total := resp.ContentLength
	var downloaded int64

	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			tmpFile.Close()
			return ctx.Err()
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := tmpFile.Write(buf[:n])
			if writeErr != nil {
				tmpFile.Close()
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
			tmpFile.Close()
			return err
		}
	}
	tmpFile.Close()

	// Track success to clean up extracted files on failure
	ffmpegPath := filepath.Join(binDir, ffmpegBinaryName())
	ffprobePath := filepath.Join(binDir, ffprobeBinaryName())
	success := false
	defer func() {
		if !success {
			os.Remove(ffmpegPath)
			os.Remove(ffprobePath)
		}
	}()

	if err := d.extractFFmpegFrom7z(tmpPath, binDir); err != nil {
		return err
	}

	d.ffmpegPath = ffmpegPath
	success = true
	return nil
}

func (d *Downloader) extractFFmpegFrom7z(archivePath, binDir string) error {
	r, err := sevenzip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open 7z archive: %w", err)
	}
	defer r.Close()

	// Files to extract (inside "ffmpeg-X.X.X-essentials_build/bin/")
	targets := map[string]string{
		"ffmpeg.exe":  ffmpegBinaryName(),
		"ffprobe.exe": ffprobeBinaryName(),
	}
	extracted := make(map[string]bool)

	for _, f := range r.File {
		basename := filepath.Base(f.Name)
		destName, needed := targets[basename]
		if !needed {
			continue
		}

		destPath := filepath.Join(binDir, destName)
		if err := d.extractFileFrom7z(f, destPath); err != nil {
			return fmt.Errorf("failed to extract %s: %w", basename, err)
		}
		extracted[basename] = true
	}

	// Verify ffmpeg was extracted (ffprobe is optional but expected)
	if !extracted["ffmpeg.exe"] {
		return fmt.Errorf("ffmpeg.exe not found in 7z archive")
	}

	return nil
}

func (d *Downloader) extractFileFrom7z(f *sevenzip.File, destPath string) error {
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(destPath)
		return err
	}

	// Make executable on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(destPath, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (d *Downloader) DownloadThumbnail(ctx context.Context, reelID string, destPath string) error {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return fmt.Errorf("yt-dlp not found")
	}

	url := buildReelURL(reelID)

	// Download thumbnail only
	args := []string{
		"--no-warnings",
		"--skip-download",
		"--write-thumbnail",
		"--convert-thumbnails", "jpg",
		"-o", strings.TrimSuffix(destPath, filepath.Ext(destPath)),
		url,
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "Private video") || strings.Contains(stderr, "Video unavailable") {
				return domain.ErrReelNotFound
			}
			if strings.Contains(stderr, "rate") || strings.Contains(stderr, "429") {
				return domain.ErrRateLimited
			}
			return fmt.Errorf("thumbnail download failed: %s", strings.TrimSpace(stderr))
		}
		return fmt.Errorf("thumbnail download failed: %w", err)
	}

	// yt-dlp adds extension, verify file exists
	expectedPath := strings.TrimSuffix(destPath, filepath.Ext(destPath)) + ".jpg"
	if _, err := os.Stat(expectedPath); err != nil {
		// Try webp as fallback
		webpPath := strings.TrimSuffix(destPath, filepath.Ext(destPath)) + ".webp"
		if _, err := os.Stat(webpPath); err == nil {
			return os.Rename(webpPath, destPath)
		}
		return fmt.Errorf("thumbnail file not found after download")
	}

	if expectedPath != destPath {
		return os.Rename(expectedPath, destPath)
	}
	return nil
}

func (d *Downloader) DownloadVideo(ctx context.Context, reelID string, destPath string) error {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return fmt.Errorf("yt-dlp not found")
	}

	url := buildReelURL(reelID)

	// Download best quality video
	args := []string{
		"--no-warnings",
		"-f", "best",
		"-o", destPath,
		url,
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "Private video") || strings.Contains(stderr, "Video unavailable") {
				return domain.ErrReelNotFound
			}
			if strings.Contains(stderr, "rate") || strings.Contains(stderr, "429") {
				return domain.ErrRateLimited
			}
			return fmt.Errorf("video download failed: %s", strings.TrimSpace(stderr))
		}
		return fmt.Errorf("video download failed: %w", err)
	}

	return nil
}

// Ensure Downloader implements interfaces
var _ ports.VideoDownloader = (*Downloader)(nil)
var _ ports.AccountFetcher = (*Downloader)(nil)
