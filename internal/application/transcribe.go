package application

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

// TranscribeOptions configures the transcription
type TranscribeOptions struct {
	Model         string
	Format        string // text, srt, json
	NoCache       bool
	Language      string // empty defaults to "auto"
	SaveAudio     bool   // Save WAV audio file
	SaveVideo     bool   // Save MP4 video file
	SaveThumbnail bool
	OutputDir     string // directory for outputs
}

// TranscribeResult contains the transcription result
type TranscribeResult struct {
	Reel          *domain.Reel
	Transcript    *domain.Transcript
	AudioPath     string // WAV audio path
	VideoPath     string // MP4 video path
	ThumbnailPath string

	// Per-asset cache status
	TranscriptFromCache bool
	AudioFromCache      bool
	VideoFromCache      bool
	ThumbnailFromCache  bool
}

// TranscribeService orchestrates the transcription process
type TranscribeService struct {
	cache       ports.CacheStore
	downloader  ports.VideoDownloader
	transcriber ports.Transcriber
	cacheTTL    time.Duration
}

// NewTranscribeService creates a new transcription service
func NewTranscribeService(
	cache ports.CacheStore,
	downloader ports.VideoDownloader,
	transcriber ports.Transcriber,
	cacheTTL time.Duration,
) *TranscribeService {
	return &TranscribeService{
		cache:       cache,
		downloader:  downloader,
		transcriber: transcriber,
		cacheTTL:    cacheTTL,
	}
}

// Transcribe processes a reel and returns its transcript
func (s *TranscribeService) Transcribe(ctx context.Context, reelID string, opts TranscribeOptions) (*TranscribeResult, error) {
	result := &TranscribeResult{}

	// Try to get cached data first
	var cached *ports.CachedItem
	if !opts.NoCache {
		var err error
		cached, err = s.cache.Get(ctx, reelID)
		if err != nil {
			cached = nil // Treat errors as cache miss
		}
	}

	// Determine what we have cached vs what we need
	cacheDir := s.cache.GetCacheDir(reelID)

	hasTranscript := cached != nil && cached.Transcript != nil
	hasAudio := cached != nil && cached.AudioPath != "" && fileExists(cached.AudioPath)
	hasVideo := cached != nil && cached.VideoPath != "" && fileExists(cached.VideoPath)
	hasThumbnail := cached != nil && cached.ThumbnailPath != "" && fileExists(cached.ThumbnailPath)

	needTranscript := !hasTranscript
	needAudio := (opts.SaveAudio || needTranscript) && !hasAudio
	needVideo := opts.SaveVideo && !hasVideo
	needThumbnail := opts.SaveThumbnail && !hasThumbnail

	// Use cached reel metadata if available
	var reel *domain.Reel
	if cached != nil && cached.Reel != nil {
		reel = cached.Reel
	}

	// Download audio if needed for transcription OR if user requested audio
	var audioPath string
	if needAudio {
		downloadResult, err := s.downloader.DownloadAudio(ctx, reelID, cacheDir)
		if err != nil {
			return nil, err
		}
		audioPath = downloadResult.AudioPath
		reel = downloadResult.Reel
	} else if hasAudio {
		audioPath = cached.AudioPath
	}

	// Transcribe if needed
	var transcript *domain.Transcript
	if needTranscript {
		model := opts.Model
		if model == "" {
			model = "small"
		}

		language := opts.Language
		if language == "" {
			language = "auto"
		}

		var err error
		transcript, err = s.transcriber.Transcribe(ctx, audioPath, ports.TranscribeOpts{
			Model:    model,
			Language: language,
		})
		if err != nil {
			return nil, err
		}
	} else {
		transcript = cached.Transcript
	}

	// Download video if needed (separate from audio)
	var videoPath string
	if needVideo {
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return nil, err
		}
		vidPath := filepath.Join(cacheDir, "video.mp4")
		if err := s.downloader.DownloadVideo(ctx, reelID, vidPath); err != nil {
			// Non-fatal - continue without video
		} else {
			videoPath = vidPath
		}
	} else if hasVideo {
		videoPath = cached.VideoPath
	}

	// Download thumbnail if needed
	var thumbnailPath string
	if needThumbnail {
		thumbPath := filepath.Join(cacheDir, "thumbnail.jpg")
		if err := s.downloader.DownloadThumbnail(ctx, reelID, thumbPath); err != nil {
			// Non-fatal - continue without thumbnail
		} else {
			thumbnailPath = thumbPath
		}
	} else if hasThumbnail {
		thumbnailPath = cached.ThumbnailPath
	}

	// Update cache with any new data
	now := time.Now()
	cacheItem := &ports.CachedItem{
		Reel:          reel,
		Transcript:    transcript,
		AudioPath:     audioPath,
		VideoPath:     videoPath,
		ThumbnailPath: thumbnailPath,
		CreatedAt:     now,
		ExpiresAt:     now.Add(s.cacheTTL),
	}

	// Preserve original timestamps if we had cached data
	if cached != nil {
		cacheItem.CreatedAt = cached.CreatedAt
	}

	_ = s.cache.Set(ctx, reelID, cacheItem)

	// Build result
	result.Reel = reel
	result.Transcript = transcript
	result.AudioPath = audioPath
	result.VideoPath = videoPath
	result.ThumbnailPath = thumbnailPath
	result.TranscriptFromCache = hasTranscript
	result.AudioFromCache = hasAudio && (opts.SaveAudio || needTranscript)
	result.VideoFromCache = hasVideo && opts.SaveVideo
	result.ThumbnailFromCache = hasThumbnail && opts.SaveThumbnail

	return result, nil
}

// fileExists checks if a file exists at the given path
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
