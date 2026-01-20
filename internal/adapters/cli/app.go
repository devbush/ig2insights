package cli

import (
	"time"

	"github.com/devbush/ig2insights/internal/adapters/cache"
	"github.com/devbush/ig2insights/internal/adapters/whisper"
	"github.com/devbush/ig2insights/internal/adapters/ytdlp"
	"github.com/devbush/ig2insights/internal/application"
	"github.com/devbush/ig2insights/internal/config"
	"github.com/devbush/ig2insights/internal/ports"
)

// App holds all application dependencies
type App struct {
	Config      *config.Config
	Cache       ports.CacheStore
	Downloader  *ytdlp.Downloader
	Transcriber *whisper.Transcriber

	TranscribeSvc *application.TranscribeService
	BrowseSvc     *application.BrowseService
	CacheSvc      *application.CacheService
}

// NewApp creates and wires up all dependencies
func NewApp() (*App, error) {
	// Ensure directories exist
	if err := config.EnsureDirs(); err != nil {
		return nil, err
	}

	// Load config
	cfg, err := config.LoadDefault()
	if err != nil {
		return nil, err
	}

	// Parse cache TTL
	ttl, err := cfg.GetCacheTTL()
	if err != nil {
		ttl = 7 * 24 * time.Hour // Default
	}

	// Create adapters
	cacheStore := cache.NewFileCache(config.CacheDir(), ttl)
	downloader := ytdlp.NewDownloader()
	transcriber := whisper.NewTranscriber("")

	// Create services
	transcribeSvc := application.NewTranscribeService(cacheStore, downloader, transcriber, ttl)
	browseSvc := application.NewBrowseService(downloader)
	cacheSvc := application.NewCacheService(cacheStore)

	return &App{
		Config:        cfg,
		Cache:         cacheStore,
		Downloader:    downloader,
		Transcriber:   transcriber,
		TranscribeSvc: transcribeSvc,
		BrowseSvc:     browseSvc,
		CacheSvc:      cacheSvc,
	}, nil
}

var globalApp *App

// GetApp returns the global app instance, creating it if needed
func GetApp() (*App, error) {
	if globalApp == nil {
		app, err := NewApp()
		if err != nil {
			return nil, err
		}
		globalApp = app
	}
	return globalApp, nil
}
