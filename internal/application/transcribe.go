package application

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

const (
	defaultModel    = "small"
	defaultLanguage = "auto"
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

// cacheState tracks what assets are available from cache
type cacheState struct {
	item         *ports.CachedItem
	hasTranscript bool
	hasAudio      bool
	hasVideo      bool
	hasThumbnail  bool
}

// Transcribe processes a reel and returns its transcript
func (s *TranscribeService) Transcribe(ctx context.Context, reelID string, opts TranscribeOptions) (*TranscribeResult, error) {
	cacheDir := s.cache.GetCacheDir(reelID)
	cache := s.loadCacheState(ctx, reelID, opts.NoCache)

	reel := s.reelFromCache(cache)
	audioPath, reel, err := s.resolveAudio(ctx, reelID, cacheDir, opts, cache, reel)
	if err != nil {
		return nil, err
	}

	transcript, err := s.resolveTranscript(ctx, audioPath, opts, cache)
	if err != nil {
		return nil, err
	}

	videoPath := s.resolveVideo(ctx, reelID, cacheDir, opts, cache)
	thumbnailPath := s.resolveThumbnail(ctx, reelID, cacheDir, opts, cache)

	s.updateCache(ctx, reelID, reel, transcript, audioPath, videoPath, thumbnailPath, cache)

	return &TranscribeResult{
		Reel:                reel,
		Transcript:          transcript,
		AudioPath:           audioPath,
		VideoPath:           videoPath,
		ThumbnailPath:       thumbnailPath,
		TranscriptFromCache: cache.hasTranscript,
		AudioFromCache:      cache.hasAudio && (opts.SaveAudio || !cache.hasTranscript),
		VideoFromCache:      cache.hasVideo && opts.SaveVideo,
		ThumbnailFromCache:  cache.hasThumbnail && opts.SaveThumbnail,
	}, nil
}

func (s *TranscribeService) loadCacheState(ctx context.Context, reelID string, noCache bool) cacheState {
	if noCache {
		return cacheState{}
	}

	item, err := s.cache.Get(ctx, reelID)
	if err != nil || item == nil {
		return cacheState{}
	}

	return cacheState{
		item:          item,
		hasTranscript: item.Transcript != nil,
		hasAudio:      item.AudioPath != "" && fileExists(item.AudioPath),
		hasVideo:      item.VideoPath != "" && fileExists(item.VideoPath),
		hasThumbnail:  item.ThumbnailPath != "" && fileExists(item.ThumbnailPath),
	}
}

func (s *TranscribeService) reelFromCache(cache cacheState) *domain.Reel {
	if cache.item != nil && cache.item.Reel != nil {
		return cache.item.Reel
	}
	return nil
}

func (s *TranscribeService) resolveAudio(
	ctx context.Context,
	reelID, cacheDir string,
	opts TranscribeOptions,
	cache cacheState,
	reel *domain.Reel,
) (string, *domain.Reel, error) {
	needTranscript := !cache.hasTranscript
	needAudio := (opts.SaveAudio || needTranscript) && !cache.hasAudio

	if !needAudio {
		if cache.hasAudio {
			return cache.item.AudioPath, reel, nil
		}
		return "", reel, nil
	}

	result, err := s.downloader.DownloadAudio(ctx, reelID, cacheDir)
	if err != nil {
		return "", nil, err
	}
	return result.AudioPath, result.Reel, nil
}

func (s *TranscribeService) resolveTranscript(
	ctx context.Context,
	audioPath string,
	opts TranscribeOptions,
	cache cacheState,
) (*domain.Transcript, error) {
	if cache.hasTranscript {
		return cache.item.Transcript, nil
	}

	model := opts.Model
	if model == "" {
		model = defaultModel
	}

	language := opts.Language
	if language == "" {
		language = defaultLanguage
	}

	return s.transcriber.Transcribe(ctx, audioPath, ports.TranscribeOpts{
		Model:    model,
		Language: language,
	})
}

func (s *TranscribeService) resolveVideo(
	ctx context.Context,
	reelID, cacheDir string,
	opts TranscribeOptions,
	cache cacheState,
) string {
	if !opts.SaveVideo {
		return ""
	}

	if cache.hasVideo {
		return cache.item.VideoPath
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return ""
	}

	videoPath := filepath.Join(cacheDir, "video.mp4")
	if err := s.downloader.DownloadVideo(ctx, reelID, videoPath); err != nil {
		return ""
	}
	return videoPath
}

func (s *TranscribeService) resolveThumbnail(
	ctx context.Context,
	reelID, cacheDir string,
	opts TranscribeOptions,
	cache cacheState,
) string {
	if !opts.SaveThumbnail {
		return ""
	}

	if cache.hasThumbnail {
		return cache.item.ThumbnailPath
	}

	thumbnailPath := filepath.Join(cacheDir, "thumbnail.jpg")
	if err := s.downloader.DownloadThumbnail(ctx, reelID, thumbnailPath); err != nil {
		return ""
	}
	return thumbnailPath
}

func (s *TranscribeService) updateCache(
	ctx context.Context,
	reelID string,
	reel *domain.Reel,
	transcript *domain.Transcript,
	audioPath, videoPath, thumbnailPath string,
	cache cacheState,
) {
	now := time.Now()
	createdAt := now
	if cache.item != nil {
		createdAt = cache.item.CreatedAt
	}

	_ = s.cache.Set(ctx, reelID, &ports.CachedItem{
		Reel:          reel,
		Transcript:    transcript,
		AudioPath:     audioPath,
		VideoPath:     videoPath,
		ThumbnailPath: thumbnailPath,
		CreatedAt:     createdAt,
		ExpiresAt:     now.Add(s.cacheTTL),
	})
}

// fileExists checks if a file exists at the given path
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
