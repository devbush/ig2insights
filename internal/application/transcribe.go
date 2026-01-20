package application

import (
	"context"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

// TranscribeOptions configures the transcription
type TranscribeOptions struct {
	Model    string
	Format   string // text, srt, json
	NoCache  bool
	Language string // empty defaults to "auto"
}

// TranscribeResult contains the transcription result
type TranscribeResult struct {
	Reel       *domain.Reel
	Transcript *domain.Transcript
	FromCache  bool
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
	// Check cache first (unless bypassed)
	if !opts.NoCache {
		cached, err := s.cache.Get(ctx, reelID)
		if err == nil {
			return &TranscribeResult{
				Reel:       cached.Reel,
				Transcript: cached.Transcript,
				FromCache:  true,
			}, nil
		}
	}

	// Download video
	cacheDir := s.cache.GetCacheDir(reelID)
	downloadResult, err := s.downloader.Download(ctx, reelID, cacheDir)
	if err != nil {
		return nil, err
	}

	// Transcribe
	model := opts.Model
	if model == "" {
		model = "small"
	}

	language := opts.Language
	if language == "" {
		language = "auto"
	}

	transcript, err := s.transcriber.Transcribe(ctx, downloadResult.VideoPath, ports.TranscribeOpts{
		Model:    model,
		Language: language,
	})
	if err != nil {
		return nil, err
	}

	// Cache result
	now := time.Now()
	cacheItem := &ports.CachedItem{
		Reel:       downloadResult.Reel,
		Transcript: transcript,
		VideoPath:  downloadResult.VideoPath,
		CreatedAt:  now,
		ExpiresAt:  now.Add(s.cacheTTL),
	}

	// Cache result (failures are non-fatal)
	_ = s.cache.Set(ctx, reelID, cacheItem)

	return &TranscribeResult{
		Reel:       downloadResult.Reel,
		Transcript: transcript,
		FromCache:  false,
	}, nil
}
