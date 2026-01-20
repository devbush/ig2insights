# FFmpeg Auto-Download Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Auto-download ffmpeg on Windows so yt-dlp can extract audio from Instagram reels.

**Architecture:** Add ffmpeg methods to the existing Downloader adapter (same pattern as whisper.cpp on Transcriber). Windows auto-downloads from gyan.dev, macOS/Linux show install instructions.

**Tech Stack:** Go, github.com/bodgit/sevenzip for 7z extraction

---

### Task 1: Add sevenzip dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add the sevenzip dependency**

Run:
```bash
go get github.com/bodgit/sevenzip
```

**Step 2: Verify dependency was added**

Run:
```bash
grep sevenzip go.mod
```
Expected: Line containing `github.com/bodgit/sevenzip`

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add bodgit/sevenzip dependency for 7z extraction

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Add ErrFFmpegNotFound to domain errors

**Files:**
- Modify: `internal/domain/errors.go`
- Test: `internal/domain/errors_test.go` (create if needed)

**Step 1: Add the error constant**

Add after `ErrCacheMiss`:

```go
	// ErrFFmpegNotFound indicates ffmpeg is not installed
	ErrFFmpegNotFound = errors.New("ffmpeg not found")
```

**Step 2: Verify file compiles**

Run:
```bash
go build ./internal/domain/...
```
Expected: No errors

**Step 3: Commit**

```bash
git add internal/domain/errors.go
git commit -m "feat(domain): add ErrFFmpegNotFound error

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 3: Add ffmpegPath field and helper functions to Downloader

**Files:**
- Modify: `internal/adapters/ytdlp/downloader.go`
- Test: `internal/adapters/ytdlp/downloader_test.go`

**Step 1: Write the failing tests**

Add to `downloader_test.go`:

```go
func TestFFmpegBinaryName(t *testing.T) {
	name := ffmpegBinaryName()
	if runtime.GOOS == "windows" {
		if name != "ffmpeg.exe" {
			t.Errorf("ffmpegBinaryName() = %q, want 'ffmpeg.exe'", name)
		}
	} else {
		if name != "ffmpeg" {
			t.Errorf("ffmpegBinaryName() = %q, want 'ffmpeg'", name)
		}
	}
}

func TestFFprobeBinaryName(t *testing.T) {
	name := ffprobeBinaryName()
	if runtime.GOOS == "windows" {
		if name != "ffprobe.exe" {
			t.Errorf("ffprobeBinaryName() = %q, want 'ffprobe.exe'", name)
		}
	} else {
		if name != "ffprobe" {
			t.Errorf("ffprobeBinaryName() = %q, want 'ffprobe'", name)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run:
```bash
go test ./internal/adapters/ytdlp/... -run "TestFFmpeg|TestFFprobe" -v
```
Expected: FAIL - undefined functions

**Step 3: Implement the code**

Add to `downloader.go` after `binaryName()`:

```go
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
```

Update Downloader struct:

```go
type Downloader struct {
	binPath    string
	ffmpegPath string
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/adapters/ytdlp/... -run "TestFFmpeg|TestFFprobe" -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/adapters/ytdlp/downloader.go internal/adapters/ytdlp/downloader_test.go
git commit -m "feat(ytdlp): add ffmpegPath field and binary name helpers

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 4: Add findFFmpeg, GetFFmpegPath, and IsFFmpegAvailable methods

**Files:**
- Modify: `internal/adapters/ytdlp/downloader.go`
- Test: `internal/adapters/ytdlp/downloader_test.go`

**Step 1: Write the failing tests**

Add to `downloader_test.go`:

```go
func TestIsFFmpegAvailable_NotInstalled(t *testing.T) {
	d := NewDownloader()
	// On a fresh system without ffmpeg in PATH or bundled, this should return false
	// We can't easily test this without mocking, so we just verify the method exists
	_ = d.IsFFmpegAvailable()
}

func TestGetFFmpegPath_Caching(t *testing.T) {
	d := NewDownloader()
	d.ffmpegPath = "/cached/path/ffmpeg"

	path := d.GetFFmpegPath()
	if path != "/cached/path/ffmpeg" {
		t.Errorf("GetFFmpegPath() should return cached path, got %q", path)
	}
}
```

**Step 2: Run tests to verify they fail**

Run:
```bash
go test ./internal/adapters/ytdlp/... -run "TestIsFFmpegAvailable|TestGetFFmpegPath" -v
```
Expected: FAIL - undefined methods

**Step 3: Implement the code**

Add to `downloader.go`:

```go
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
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/adapters/ytdlp/... -run "TestIsFFmpegAvailable|TestGetFFmpegPath" -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/adapters/ytdlp/downloader.go internal/adapters/ytdlp/downloader_test.go
git commit -m "feat(ytdlp): add GetFFmpegPath and IsFFmpegAvailable methods

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 5: Add getFFmpegDownloadURL and FFmpegInstructions methods

**Files:**
- Modify: `internal/adapters/ytdlp/downloader.go`
- Test: `internal/adapters/ytdlp/downloader_test.go`

**Step 1: Write the failing tests**

Add to `downloader_test.go`:

```go
func TestGetFFmpegDownloadURL(t *testing.T) {
	d := NewDownloader()
	url := d.getFFmpegDownloadURL()

	if runtime.GOOS == "windows" {
		expected := "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.7z"
		if url != expected {
			t.Errorf("getFFmpegDownloadURL() = %q, want %q", url, expected)
		}
	} else {
		if url != "" {
			t.Errorf("getFFmpegDownloadURL() should be empty on non-Windows, got %q", url)
		}
	}
}

func TestFFmpegInstructions(t *testing.T) {
	d := NewDownloader()
	instructions := d.FFmpegInstructions()

	switch runtime.GOOS {
	case "windows":
		if instructions != "" {
			t.Errorf("FFmpegInstructions() should be empty on Windows (auto-download), got %q", instructions)
		}
	case "darwin":
		if !strings.Contains(instructions, "brew install ffmpeg") {
			t.Errorf("FFmpegInstructions() should mention brew, got %q", instructions)
		}
	default:
		if !strings.Contains(instructions, "apt") && !strings.Contains(instructions, "dnf") {
			t.Errorf("FFmpegInstructions() should mention apt or dnf, got %q", instructions)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run:
```bash
go test ./internal/adapters/ytdlp/... -run "TestGetFFmpegDownloadURL|TestFFmpegInstructions" -v
```
Expected: FAIL - undefined methods

**Step 3: Implement the code**

Add to `downloader.go`:

```go
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
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/adapters/ytdlp/... -run "TestGetFFmpegDownloadURL|TestFFmpegInstructions" -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/adapters/ytdlp/downloader.go internal/adapters/ytdlp/downloader_test.go
git commit -m "feat(ytdlp): add getFFmpegDownloadURL and FFmpegInstructions

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 6: Add extractFFmpegFrom7z method

**Files:**
- Modify: `internal/adapters/ytdlp/downloader.go`
- Test: `internal/adapters/ytdlp/downloader_test.go`

**Step 1: Write the failing tests**

Add imports to test file:
```go
import (
	"archive/zip"
	// ... existing imports
)
```

Note: We can't easily create a real 7z file in tests, so we'll test the error case and rely on integration testing for success.

Add to `downloader_test.go`:

```go
func TestExtractFFmpegFrom7z_InvalidArchive(t *testing.T) {
	tmpDir := t.TempDir()
	d := NewDownloader()

	// Create an invalid file
	invalidPath := filepath.Join(tmpDir, "invalid.7z")
	os.WriteFile(invalidPath, []byte("not a 7z file"), 0644)

	err := d.extractFFmpegFrom7z(invalidPath, tmpDir)
	if err == nil {
		t.Error("extractFFmpegFrom7z() should fail on invalid archive")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/adapters/ytdlp/... -run "TestExtractFFmpegFrom7z" -v
```
Expected: FAIL - undefined method

**Step 3: Implement the code**

Add import:
```go
import (
	// ... existing imports
	"github.com/bodgit/sevenzip"
)
```

Add to `downloader.go`:

```go
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
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/adapters/ytdlp/... -run "TestExtractFFmpegFrom7z" -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/adapters/ytdlp/downloader.go internal/adapters/ytdlp/downloader_test.go
git commit -m "feat(ytdlp): add extractFFmpegFrom7z method

Uses bodgit/sevenzip to extract ffmpeg.exe and ffprobe.exe
from the gyan.dev essentials build.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 7: Add InstallFFmpeg method

**Files:**
- Modify: `internal/adapters/ytdlp/downloader.go`
- Test: `internal/adapters/ytdlp/downloader_test.go`

**Step 1: Write the failing tests**

Add to `downloader_test.go`:

```go
func TestInstallFFmpeg_NonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping non-Windows test on Windows")
	}

	d := NewDownloader()
	err := d.InstallFFmpeg(context.Background(), nil)

	if err == nil {
		t.Error("InstallFFmpeg() should return error on non-Windows")
	}
	if !strings.Contains(err.Error(), "no prebuilt") {
		t.Errorf("InstallFFmpeg() error should mention 'no prebuilt', got: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/adapters/ytdlp/... -run "TestInstallFFmpeg" -v
```
Expected: FAIL - undefined method

**Step 3: Implement the code**

Add to `downloader.go`:

```go
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
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/adapters/ytdlp/... -run "TestInstallFFmpeg" -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/adapters/ytdlp/downloader.go internal/adapters/ytdlp/downloader_test.go
git commit -m "feat(ytdlp): add InstallFFmpeg method

Downloads ffmpeg-release-essentials.7z from gyan.dev and extracts
ffmpeg.exe + ffprobe.exe to ~/.ig2insights/bin/ on Windows.
Returns error with instructions on macOS/Linux.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 8: Update Download method to extract audio

**Files:**
- Modify: `internal/adapters/ytdlp/downloader.go`

**Step 1: Update the Download method**

Modify the `Download` function to:
1. Check if ffmpeg is available before downloading
2. Add audio extraction flags to yt-dlp

Replace the args and output template section:

```go
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
	// Extract audio only since whisper only supports audio formats
	args := []string{
		"--no-warnings",
		"--print-json",
		"-x",                    // Extract audio
		"--audio-format", "wav", // Convert to wav (whisper-compatible)
		"-o", outputTemplate,
		url,
	}

	// ... rest of the function stays the same but update:
	// - Change "video.*" glob to "audio.*"
	// - Change videoPath variable to audioPath
```

**Step 2: Update the glob pattern and variable names**

Change:
- `filepath.Glob(filepath.Join(destDir, "video.*"))` → `filepath.Glob(filepath.Join(destDir, "audio.*"))`
- `videoPath` → `audioPath`
- Return `audioPath` in the result

**Step 3: Verify it compiles**

Run:
```bash
go build ./internal/adapters/ytdlp/...
```
Expected: No errors

**Step 4: Commit**

```bash
git add internal/adapters/ytdlp/downloader.go
git commit -m "feat(ytdlp): update Download to extract audio for whisper

- Check ffmpeg availability before download
- Add -x and --audio-format wav flags for audio extraction
- Rename output to audio.wav for whisper compatibility

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 9: Update ports.VideoDownloader interface

**Files:**
- Modify: `internal/ports/downloader.go`

**Step 1: Add ffmpeg methods to interface**

Add after `Update`:

```go
	// IsFFmpegAvailable checks if ffmpeg is installed
	IsFFmpegAvailable() bool

	// GetFFmpegPath returns path to ffmpeg binary
	GetFFmpegPath() string

	// InstallFFmpeg downloads and installs ffmpeg (Windows only)
	InstallFFmpeg(ctx context.Context, progress func(downloaded, total int64)) error

	// FFmpegInstructions returns platform-specific install instructions
	FFmpegInstructions() string
```

**Step 2: Verify interface is satisfied**

Run:
```bash
go build ./...
```
Expected: No errors (Downloader already implements these methods)

**Step 3: Commit**

```bash
git add internal/ports/downloader.go
git commit -m "feat(ports): add ffmpeg methods to VideoDownloader interface

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 10: Update deps command

**Files:**
- Modify: `internal/adapters/cli/deps.go`

**Step 1: Update Short description**

Change:
```go
Short: "Manage dependencies (yt-dlp, whisper.cpp)",
```
To:
```go
Short: "Manage dependencies (yt-dlp, whisper.cpp, ffmpeg)",
```

**Step 2: Update runDepsStatus to show ffmpeg**

Add after whisper.cpp section:

```go
	// ffmpeg
	if app.Downloader.IsFFmpegAvailable() {
		path := app.Downloader.GetFFmpegPath()
		fmt.Printf("  ffmpeg:        installed (%s)\n", path)
	} else {
		fmt.Println("  ffmpeg:        not found")
	}
```

**Step 3: Update runDepsInstall to install ffmpeg**

Add after whisper.cpp installation:

```go
	// Install ffmpeg
	if app.Downloader.IsFFmpegAvailable() {
		fmt.Println("ffmpeg is already installed")
	} else {
		instructions := app.Downloader.FFmpegInstructions()
		if instructions != "" {
			fmt.Println(instructions)
		} else {
			fmt.Println("Installing ffmpeg...")
			if err := app.Downloader.InstallFFmpeg(ctx, progress); err != nil {
				return fmt.Errorf("failed to install ffmpeg: %w", err)
			}
			fmt.Println("\nffmpeg installed")
		}
	}
```

**Step 4: Verify it compiles**

Run:
```bash
go build ./internal/adapters/cli/...
```
Expected: No errors

**Step 5: Commit**

```bash
git add internal/adapters/cli/deps.go
git commit -m "feat(cli): add ffmpeg to deps status and install commands

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 11: Update root command to auto-install ffmpeg

**Files:**
- Modify: `internal/adapters/cli/root.go`

**Step 1: Add ffmpeg check in runTranscribe**

Add after the whisper.cpp check block (around line 144):

```go
	// Check ffmpeg
	if !app.Downloader.IsFFmpegAvailable() {
		instructions := app.Downloader.FFmpegInstructions()
		if instructions != "" {
			return errors.New(instructions)
		}
		fmt.Println("ffmpeg not found. Installing...")
		if err := app.Downloader.InstallFFmpeg(context.Background(), printProgress); err != nil {
			return fmt.Errorf("failed to install ffmpeg: %w", err)
		}
		fmt.Println("\n✓ ffmpeg installed")
	}
```

**Step 2: Verify it compiles**

Run:
```bash
go build ./cmd/ig2insights
```
Expected: No errors

**Step 3: Commit**

```bash
git add internal/adapters/cli/root.go
git commit -m "feat(cli): auto-install ffmpeg before transcribing

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 12: Run all tests and verify

**Step 1: Run all tests**

Run:
```bash
go test ./... -v
```
Expected: All tests PASS

**Step 2: Build and test manually**

Run:
```bash
go build -o ig2insights.exe ./cmd/ig2insights
./ig2insights.exe deps status
```
Expected: Shows yt-dlp, whisper.cpp, and ffmpeg status

**Step 3: Test ffmpeg installation (Windows only)**

If ffmpeg not installed:
```bash
./ig2insights.exe deps install
```
Expected: Downloads and installs ffmpeg

**Step 4: Test transcription end-to-end**

```bash
./ig2insights.exe DTgfo7WkpNq
```
Expected: Successfully transcribes the reel

---

### Summary

**Files created:** None

**Files modified:**
- `go.mod` - Add sevenzip dependency
- `internal/domain/errors.go` - Add ErrFFmpegNotFound
- `internal/adapters/ytdlp/downloader.go` - Add all ffmpeg methods
- `internal/adapters/ytdlp/downloader_test.go` - Add ffmpeg tests
- `internal/ports/downloader.go` - Add ffmpeg interface methods
- `internal/adapters/cli/deps.go` - Add ffmpeg to deps command
- `internal/adapters/cli/root.go` - Auto-install ffmpeg before transcribe
