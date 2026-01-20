package whisper

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAvailableModels(t *testing.T) {
	tr := NewTranscriber("")
	models := tr.AvailableModels()

	if len(models) != 5 {
		t.Errorf("AvailableModels() returned %d models, want 5", len(models))
	}

	// Check that "small" exists
	found := false
	for _, m := range models {
		if m.Name == "small" {
			found = true
			if m.Size == 0 {
				t.Error("small model has zero size")
			}
		}
	}
	if !found {
		t.Error("small model not found in AvailableModels()")
	}
}

func TestModelURL(t *testing.T) {
	url := modelURL("small")
	expected := "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin"

	if url != expected {
		t.Errorf("modelURL(small) = %s, want %s", url, expected)
	}
}

func TestModelURLAllModels(t *testing.T) {
	tests := []struct {
		model    string
		expected string
	}{
		{"tiny", "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin"},
		{"base", "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin"},
		{"small", "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin"},
		{"medium", "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium.bin"},
		{"large", "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large.bin"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			url := modelURL(tt.model)
			if url != tt.expected {
				t.Errorf("modelURL(%s) = %s, want %s", tt.model, url, tt.expected)
			}
		})
	}
}

func TestIsModelDownloaded(t *testing.T) {
	tmpDir := t.TempDir()
	tr := NewTranscriber(tmpDir)

	// Model not downloaded
	if tr.IsModelDownloaded("small") {
		t.Error("IsModelDownloaded() = true for non-existent model")
	}

	// Create fake model file
	modelPath := filepath.Join(tmpDir, "ggml-small.bin")
	if err := os.WriteFile(modelPath, []byte("fake model"), 0644); err != nil {
		t.Fatalf("failed to create test model file: %v", err)
	}

	// Model now exists
	if !tr.IsModelDownloaded("small") {
		t.Error("IsModelDownloaded() = false for existing model")
	}
}

func TestDeleteModel(t *testing.T) {
	tmpDir := t.TempDir()
	tr := NewTranscriber(tmpDir)

	// Create fake model file
	modelPath := filepath.Join(tmpDir, "ggml-small.bin")
	if err := os.WriteFile(modelPath, []byte("fake model"), 0644); err != nil {
		t.Fatalf("failed to create test model file: %v", err)
	}

	// Verify file exists
	if !tr.IsModelDownloaded("small") {
		t.Fatal("model should exist before deletion")
	}

	// Delete the model
	if err := tr.DeleteModel("small"); err != nil {
		t.Errorf("DeleteModel() returned error: %v", err)
	}

	// Verify file is deleted
	if tr.IsModelDownloaded("small") {
		t.Error("model should not exist after deletion")
	}
}

func TestDeleteModelNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	tr := NewTranscriber(tmpDir)

	// Trying to delete non-existent model should return error
	err := tr.DeleteModel("small")
	if err == nil {
		t.Error("DeleteModel() should return error for non-existent model")
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"00:00:00,000", 0.0},
		{"00:00:01,000", 1.0},
		{"00:00:01,500", 1.5},
		{"00:01:00,000", 60.0},
		{"01:00:00,000", 3600.0},
		{"01:30:45,123", 5445.123},
		{"00:00:00.500", 0.5}, // Period instead of comma
		{"invalid", 0.0},      // Invalid format returns 0
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseTimestamp(tt.input)
			if result != tt.expected {
				t.Errorf("parseTimestamp(%s) = %f, want %f", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDownloadModelUnknown(t *testing.T) {
	tmpDir := t.TempDir()
	tr := NewTranscriber(tmpDir)

	err := tr.DownloadModel(context.Background(), "unknown-model", nil)
	if err == nil {
		t.Error("DownloadModel() should return error for unknown model")
	}
}

func TestAvailableModelsDownloadedStatus(t *testing.T) {
	tmpDir := t.TempDir()
	tr := NewTranscriber(tmpDir)

	// Create a fake model file
	modelPath := filepath.Join(tmpDir, "ggml-tiny.bin")
	if err := os.WriteFile(modelPath, []byte("fake model"), 0644); err != nil {
		t.Fatalf("failed to create test model file: %v", err)
	}

	models := tr.AvailableModels()

	for _, m := range models {
		if m.Name == "tiny" {
			if !m.Downloaded {
				t.Error("tiny model should show as downloaded")
			}
		} else {
			if m.Downloaded {
				t.Errorf("%s model should not show as downloaded", m.Name)
			}
		}
	}
}

func TestModelPath(t *testing.T) {
	tmpDir := t.TempDir()
	tr := NewTranscriber(tmpDir)

	path := tr.modelPath("small")
	expected := filepath.Join(tmpDir, "ggml-small.bin")

	if path != expected {
		t.Errorf("modelPath(small) = %s, want %s", path, expected)
	}
}

func TestNewTranscriberDefaultModelsDir(t *testing.T) {
	// Test that empty modelsDir uses default
	tr := NewTranscriber("")
	if tr.modelsDir == "" {
		t.Error("modelsDir should not be empty when using default")
	}
}

func TestNewTranscriberCustomModelsDir(t *testing.T) {
	customDir := "/custom/models/dir"
	tr := NewTranscriber(customDir)
	if tr.modelsDir != customDir {
		t.Errorf("modelsDir = %s, want %s", tr.modelsDir, customDir)
	}
}
