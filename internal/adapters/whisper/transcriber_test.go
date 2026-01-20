package whisper

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestParseWhisperJSON(t *testing.T) {
	tmpDir := t.TempDir()
	tr := NewTranscriber(tmpDir)

	jsonContent := `{
        "transcription": [
            {"timestamps": {"from": "00:00:00,000", "to": "00:00:02,500"}, "text": "Hello world"},
            {"timestamps": {"from": "00:00:02,500", "to": "00:00:05,000"}, "text": "Test segment"}
        ]
    }`

	jsonPath := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(jsonPath, []byte(jsonContent), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := tr.parseWhisperJSON(jsonPath, "small")
	if err != nil {
		t.Fatalf("parseWhisperJSON failed: %v", err)
	}

	if len(result.Segments) != 2 {
		t.Errorf("expected 2 segments, got %d", len(result.Segments))
	}

	if result.Text != "Hello world Test segment" {
		t.Errorf("expected combined text, got %q", result.Text)
	}

	if result.Model != "small" {
		t.Errorf("expected model 'small', got %q", result.Model)
	}

	// Check first segment
	if result.Segments[0].Start != 0.0 {
		t.Errorf("segment[0].Start = %f, want 0.0", result.Segments[0].Start)
	}
	if result.Segments[0].End != 2.5 {
		t.Errorf("segment[0].End = %f, want 2.5", result.Segments[0].End)
	}
	if result.Segments[0].Text != "Hello world" {
		t.Errorf("segment[0].Text = %q, want 'Hello world'", result.Segments[0].Text)
	}
}

func TestParseWhisperJSON_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	tr := NewTranscriber(tmpDir)

	_, err := tr.parseWhisperJSON(filepath.Join(tmpDir, "nonexistent.json"), "small")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestParseWhisperJSON_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	tr := NewTranscriber(tmpDir)

	jsonPath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(jsonPath, []byte("not valid json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := tr.parseWhisperJSON(jsonPath, "small")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFindWhisperBinary_NotFound(t *testing.T) {
	tr := NewTranscriber(t.TempDir())
	// This test will return "" if whisper is not installed, which is expected behavior
	path := tr.findWhisperBinary()
	// We can't assert much here since it depends on system state
	// Just verify it doesn't panic
	_ = path
}

func TestWhisperBinaryName(t *testing.T) {
	name := whisperBinaryName()
	if runtime.GOOS == "windows" {
		if name != "whisper.exe" {
			t.Errorf("whisperBinaryName() = %s, want whisper.exe", name)
		}
	} else {
		if name != "whisper" {
			t.Errorf("whisperBinaryName() = %s, want whisper", name)
		}
	}
}

func TestIsAvailable_NotInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	tr := NewTranscriber(tmpDir)
	// With no binary in bundled location or PATH, should return false
	// (unless whisper is actually installed on the system)
	_ = tr.IsAvailable() // Just verify it doesn't panic
}

func TestGetBinaryPath_Caching(t *testing.T) {
	t.Run("returns cached path", func(t *testing.T) {
		tr := &Transcriber{binPath: "/cached/path/whisper"}
		path1 := tr.GetBinaryPath()
		path2 := tr.GetBinaryPath()
		if path1 != "/cached/path/whisper" || path2 != path1 {
			t.Errorf("GetBinaryPath() didn't return cached path")
		}
	})
}

func TestGetBinaryPath_Bundled(t *testing.T) {
	// Create a temp bin directory with a fake whisper binary
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create fake binary
	binaryPath := filepath.Join(binDir, whisperBinaryName())
	if err := os.WriteFile(binaryPath, []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}

	// We can't easily test this without mocking config.BinDir()
	// So we just test that the methods exist and don't panic
	tr := NewTranscriber(tmpDir)
	_ = tr.GetBinaryPath()
	_ = tr.IsAvailable()
}

func TestGetDownloadURL(t *testing.T) {
	tr := NewTranscriber(t.TempDir())
	url := tr.getDownloadURL()

	if runtime.GOOS == "windows" {
		expected := "https://github.com/ggerganov/whisper.cpp/releases/latest/download/whisper-bin-x64.zip"
		if url != expected {
			t.Errorf("getDownloadURL() = %s, want %s", url, expected)
		}
	} else {
		// Non-Windows should return empty string
		if url != "" {
			t.Errorf("getDownloadURL() on %s should return empty, got %s", runtime.GOOS, url)
		}
	}
}

func TestInstallationInstructions(t *testing.T) {
	tr := NewTranscriber(t.TempDir())
	instructions := tr.InstallationInstructions()

	if runtime.GOOS == "windows" {
		if instructions != "" {
			t.Error("Windows should have no installation instructions (auto-download available)")
		}
	} else if runtime.GOOS == "darwin" {
		if !strings.Contains(instructions, "brew install") {
			t.Error("macOS instructions should mention brew")
		}
	} else {
		if !strings.Contains(instructions, "git clone") {
			t.Error("Linux instructions should mention git clone")
		}
	}
}

func TestInstall_NonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping non-Windows test on Windows")
	}

	tr := NewTranscriber(t.TempDir())
	err := tr.Install(context.Background(), nil)

	if err == nil {
		t.Error("Install() should return error on non-Windows")
	}
	if !strings.Contains(err.Error(), "no prebuilt") {
		t.Errorf("Install() error should mention 'no prebuilt', got: %v", err)
	}
}

func TestExtractWhisperFromZip(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	os.MkdirAll(binDir, 0755)
	tr := NewTranscriber(tmpDir)

	// Create a test zip file with whisper-cli.exe and DLLs
	zipPath := filepath.Join(tmpDir, "test.zip")

	// Create zip
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zipWriter := zip.NewWriter(zipFile)

	// Add whisper-cli.exe to zip (in Release/ directory like the real zip)
	files := map[string]string{
		"Release/whisper-cli.exe": "fake whisper binary",
		"Release/ggml-base.dll":   "fake dll 1",
		"Release/ggml-cpu.dll":    "fake dll 2",
		"Release/ggml.dll":        "fake dll 3",
		"Release/whisper.dll":     "fake dll 4",
	}
	for name, content := range files {
		w, err := zipWriter.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(content))
	}
	zipWriter.Close()
	zipFile.Close()

	// Extract
	err = tr.extractWhisperFromZip(zipPath, binDir)
	if err != nil {
		t.Fatalf("extractWhisperFromZip() failed: %v", err)
	}

	// Verify whisper binary exists (renamed to whisper.exe on Windows)
	whisperPath := filepath.Join(binDir, whisperBinaryName())
	content, err := os.ReadFile(whisperPath)
	if err != nil {
		t.Fatalf("failed to read extracted whisper binary: %v", err)
	}
	if string(content) != "fake whisper binary" {
		t.Errorf("extracted content = %q, want 'fake whisper binary'", content)
	}

	// Verify DLLs exist
	for _, dll := range []string{"ggml-base.dll", "ggml-cpu.dll", "ggml.dll", "whisper.dll"} {
		if _, err := os.Stat(filepath.Join(binDir, dll)); os.IsNotExist(err) {
			t.Errorf("expected %s to be extracted", dll)
		}
	}
}

func TestExtractWhisperFromZip_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	os.MkdirAll(binDir, 0755)
	tr := NewTranscriber(tmpDir)

	// Create a zip without whisper-cli.exe
	zipPath := filepath.Join(tmpDir, "test.zip")

	zipFile, _ := os.Create(zipPath)
	zipWriter := zip.NewWriter(zipFile)
	w, _ := zipWriter.Create("other.txt")
	w.Write([]byte("not the binary"))
	zipWriter.Close()
	zipFile.Close()

	err := tr.extractWhisperFromZip(zipPath, binDir)
	if err == nil {
		t.Error("extractWhisperFromZip() should fail when whisper-cli.exe not in zip")
	}
}
