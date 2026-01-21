package ports

import (
	"context"

	"github.com/devbush/ig2insights/internal/domain"
)

// Model represents a Whisper model available for transcription.
type Model struct {
	Name        string
	Size        int64 // size in bytes
	Description string
	Downloaded  bool
}

// TranscribeOpts configures transcription behavior.
type TranscribeOpts struct {
	Model    string
	Language string // empty string enables auto-detection
}

// Transcriber handles speech-to-text conversion.
type Transcriber interface {
	// Transcribe converts an audio/video file to a transcript.
	Transcribe(ctx context.Context, videoPath string, opts TranscribeOpts) (*domain.Transcript, error)

	// AvailableModels returns all models that can be used for transcription.
	AvailableModels() []Model

	// IsModelDownloaded checks if a model is available locally.
	IsModelDownloaded(model string) bool

	// DownloadModel downloads a model, reporting progress via callback.
	DownloadModel(ctx context.Context, model string, progress func(downloaded, total int64)) error

	// DeleteModel removes a downloaded model from local storage.
	DeleteModel(model string) error
}
