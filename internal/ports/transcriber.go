package ports

import (
	"context"

	"github.com/devbush/ig2insights/internal/domain"
)

// Model represents a Whisper model
type Model struct {
	Name        string
	Size        int64 // bytes
	Description string
	Downloaded  bool
}

// TranscribeOpts configures transcription behavior
type TranscribeOpts struct {
	Model    string
	Language string // empty for auto-detect
}

// Transcriber handles speech-to-text conversion
type Transcriber interface {
	// Transcribe converts audio/video file to transcript
	Transcribe(ctx context.Context, videoPath string, opts TranscribeOpts) (*domain.Transcript, error)

	// AvailableModels returns list of available models
	AvailableModels() []Model

	// IsModelDownloaded checks if a model is available locally
	IsModelDownloaded(model string) bool

	// DownloadModel downloads a model with progress callback
	DownloadModel(ctx context.Context, model string, progress func(downloaded, total int64)) error

	// DeleteModel removes a downloaded model
	DeleteModel(model string) error
}
