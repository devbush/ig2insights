package ytdlp

import (
	"context"
	"encoding/json"
	"errors"
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

const (
	instagramReelURLFormat = "https://www.instagram.com/p/%s/"
	instagramReelsURLFormat = "https://www.instagram.com/%s/reels/"
	ytdlpDownloadBase      = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/"
	ffmpegWindowsURL       = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.7z"
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

// executableName appends .exe suffix on Windows
func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func binaryName() string {
	return executableName("yt-dlp")
}

func ffmpegBinaryName() string {
	return executableName("ffmpeg")
}

func ffprobeBinaryName() string {
	return executableName("ffprobe")
}

func buildReelURL(reelID string) string {
	return fmt.Sprintf(instagramReelURLFormat, reelID)
}

func buildReelsURL(username string) string {
	return fmt.Sprintf(instagramReelsURLFormat, username)
}

// detectYtdlpError converts yt-dlp stderr messages to domain errors
func detectYtdlpError(err error) error {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return nil
	}

	stderr := string(exitErr.Stderr)

	if strings.Contains(stderr, "Private video") || strings.Contains(stderr, "Video unavailable") {
		return domain.ErrReelNotFound
	}
	if strings.Contains(stderr, "not found") || strings.Contains(stderr, "404") {
		return domain.ErrAccountNotFound
	}
	if strings.Contains(stderr, "rate") || strings.Contains(stderr, "429") {
		return domain.ErrRateLimited
	}
	if strings.Contains(stderr, "Unable to extract data") || strings.Contains(stderr, "Unsupported URL") {
		return domain.ErrInstagramScrapingBlocked
	}

	return nil
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
		return ffmpegWindowsURL
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

func (d *Downloader) DownloadAudio(ctx context.Context, reelID string, destDir string) (*ports.DownloadResult, error) {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return nil, fmt.Errorf("yt-dlp not found; run 'ig2insights deps install'")
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
		if domainErr := detectYtdlpError(err); domainErr != nil {
			return nil, domainErr
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("failed to run yt-dlp: %s", exitErr.Stderr)
		}
		return nil, fmt.Errorf("failed to run yt-dlp: %w", err)
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
				AudioPath: matches[0],
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
		AudioPath: audioPath,
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

	destPath := filepath.Join(binDir, binaryName())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.getDownloadURL(), nil)
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

	success := false
	defer func() {
		out.Close()
		if !success {
			os.Remove(destPath)
		}
	}()

	if err := downloadWithProgress(ctx, resp.Body, out, resp.ContentLength, progress); err != nil {
		return err
	}

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
	switch runtime.GOOS {
	case "windows":
		return ytdlpDownloadBase + "yt-dlp.exe"
	case "darwin":
		return ytdlpDownloadBase + "yt-dlp_macos"
	default:
		return ytdlpDownloadBase + "yt-dlp"
	}
}

func (d *Downloader) Update(ctx context.Context) error {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return fmt.Errorf("yt-dlp not installed; run 'ig2insights deps install'")
	}

	cmd := exec.CommandContext(ctx, binPath, "-U")
	return cmd.Run()
}

func (d *Downloader) GetAccount(ctx context.Context, username string) (*domain.Account, error) {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return nil, fmt.Errorf("yt-dlp not found; run 'ig2insights deps install'")
	}

	url := buildReelsURL(username)
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
		if domainErr := detectYtdlpError(err); domainErr != nil {
			return nil, domainErr
		}
		return nil, fmt.Errorf("failed to fetch account: %w", err)
	}

	// Count entries to estimate reel count.
	// ReelCount will be 0 or 1 due to -I 1:1 limiting to the first item.
	// A full count would require fetching the entire playlist which is slow.
	trimmed := strings.TrimSpace(string(output))
	reelCount := 0
	if trimmed != "" {
		reelCount = len(strings.Split(trimmed, "\n"))
	}

	return &domain.Account{
		Username:  username,
		ReelCount: reelCount,
	}, nil
}

func (d *Downloader) ListReels(ctx context.Context, username string, sortOrder domain.SortOrder, limit int) ([]*domain.Reel, error) {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return nil, fmt.Errorf("yt-dlp not found; run 'ig2insights deps install'")
	}

	url := buildReelsURL(username)
	args := []string{
		"--no-warnings",
		"--flat-playlist",
		"--print-json",
		"-I", fmt.Sprintf("1:%d", limit),
		url,
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if domainErr := detectYtdlpError(err); domainErr != nil {
			return nil, domainErr
		}
		return nil, fmt.Errorf("failed to list reels: %w", err)
	}

	reels := parseReelsFromOutput(output)

	if sortOrder == domain.SortMostViewed {
		sortByViews(reels)
	}

	return reels, nil
}

// reelInfo represents the JSON structure returned by yt-dlp for a reel
type reelInfo struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Uploader     string  `json:"uploader"`
	Duration     float64 `json:"duration"`
	ViewCount    int64   `json:"view_count"`
	LikeCount    int64   `json:"like_count"`
	CommentCount int64   `json:"comment_count"`
	UploadDate   string  `json:"upload_date"` // YYYYMMDD format
	Timestamp    int64   `json:"timestamp"`   // Unix timestamp
}

func parseReelsFromOutput(output []byte) []*domain.Reel {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	reels := make([]*domain.Reel, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		var info reelInfo
		if err := json.Unmarshal([]byte(line), &info); err != nil {
			continue
		}

		reels = append(reels, &domain.Reel{
			ID:              info.ID,
			Author:          info.Uploader,
			Title:           info.Title,
			DurationSeconds: int(info.Duration),
			ViewCount:       info.ViewCount,
			LikeCount:       info.LikeCount,
			CommentCount:    info.CommentCount,
			UploadedAt:      parseUploadTime(info.Timestamp, info.UploadDate),
			FetchedAt:       time.Now(),
		})
	}

	return reels
}

func parseUploadTime(timestamp int64, uploadDate string) time.Time {
	if timestamp > 0 {
		return time.Unix(timestamp, 0)
	}
	if uploadDate != "" {
		if t, err := time.Parse("20060102", uploadDate); err == nil {
			return t
		}
	}
	return time.Time{}
}

// downloadWithProgress downloads a file with context cancellation and progress reporting.
// Returns the total bytes downloaded. The caller is responsible for closing the writer.
func downloadWithProgress(ctx context.Context, body io.Reader, w io.Writer, total int64, progress func(downloaded, total int64)) error {
	buf := make([]byte, 32*1024)
	var downloaded int64

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
			if progress != nil {
				progress(downloaded, total)
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
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

	tmpFile, err := os.CreateTemp("", "ffmpeg-*.7z")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if err := downloadWithProgress(ctx, resp.Body, tmpFile, resp.ContentLength, progress); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

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
		if domainErr := detectYtdlpError(err); domainErr != nil {
			return domainErr
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("failed to download thumbnail: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return fmt.Errorf("failed to download thumbnail: %w", err)
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

	// Check for ffmpeg (needed for merging video+audio)
	if !d.IsFFmpegAvailable() {
		return domain.ErrFFmpegNotFound
	}

	url := buildReelURL(reelID)

	// Download best video+audio combined, fallback to best single stream
	args := []string{
		"--no-warnings",
		"-f", "bv*+ba/b",
		"--merge-output-format", "mp4",
		"-o", destPath,
		url,
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	if err := cmd.Run(); err != nil {
		if domainErr := detectYtdlpError(err); domainErr != nil {
			return domainErr
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("failed to download video: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return fmt.Errorf("failed to download video: %w", err)
	}

	return nil
}

// Ensure Downloader implements interfaces
var _ ports.VideoDownloader = (*Downloader)(nil)
var _ ports.AccountFetcher = (*Downloader)(nil)
