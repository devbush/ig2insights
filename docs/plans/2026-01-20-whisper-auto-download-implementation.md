# Whisper.cpp Auto-Download Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add auto-download functionality for whisper.cpp binary, matching the yt-dlp pattern.

**Architecture:** Add `Install()`, `IsAvailable()`, `GetBinaryPath()` methods to Transcriber. On Windows, download and extract from GitHub releases. On macOS/Linux, detect PATH or show installation instructions.

**Tech Stack:** Go, archive/zip, net/http

---

## Task 1: Add binPath field and whisperBinaryName function

**Files:**
- Modify: `internal/adapters/whisper/transcriber.go:32-43`
- Test: `internal/adapters/whisper/transcriber_test.go`

**Step 1: Write the failing test**

Add to `transcriber_test.go`:

```go
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
```

Add `"runtime"` to imports.

**Step 2: Run test to verify it fails**

Run: `cd /c/Users/nlafo/dev/ig2insights && go test ./internal/adapters/whisper/... -run TestWhisperBinaryName -v`
Expected: FAIL with "undefined: whisperBinaryName"

**Step 3: Write minimal implementation**

In `transcriber.go`, add `binPath` field to struct and add helper function:

```go
// Transcriber implements ports.Transcriber using whisper.cpp
type Transcriber struct {
	modelsDir string
	binPath   string
}

func whisperBinaryName() string {
	if runtime.GOOS == "windows" {
		return "whisper.exe"
	}
	return "whisper"
}
```

**Step 4: Run test to verify it passes**

Run: `cd /c/Users/nlafo/dev/ig2insights && go test ./internal/adapters/whisper/... -run TestWhisperBinaryName -v`
Expected: PASS

**Step 5: Commit**

```bash
cd /c/Users/nlafo/dev/ig2insights
git add internal/adapters/whisper/transcriber.go internal/adapters/whisper/transcriber_test.go
git commit -m "feat(whisper): add binPath field and whisperBinaryName helper"
```

---

## Task 2: Add GetBinaryPath and IsAvailable methods

**Files:**
- Modify: `internal/adapters/whisper/transcriber.go`
- Test: `internal/adapters/whisper/transcriber_test.go`

**Step 1: Write the failing tests**

Add to `transcriber_test.go`:

```go
func TestIsAvailable_NotInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	tr := NewTranscriber(tmpDir)
	// With no binary in bundled location or PATH, should return false
	// (unless whisper is actually installed on the system)
	_ = tr.IsAvailable() // Just verify it doesn't panic
}

func TestGetBinaryPath_Bundled(t *testing.T) {
	// Create a temp bin directory with a fake whisper binary
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	os.MkdirAll(binDir, 0755)

	// Create fake binary
	binaryPath := filepath.Join(binDir, whisperBinaryName())
	os.WriteFile(binaryPath, []byte("fake"), 0755)

	// We can't easily test this without mocking config.BinDir()
	// So we just test that the methods exist and don't panic
	tr := NewTranscriber(tmpDir)
	_ = tr.GetBinaryPath()
	_ = tr.IsAvailable()
}
```

**Step 2: Run test to verify it fails**

Run: `cd /c/Users/nlafo/dev/ig2insights && go test ./internal/adapters/whisper/... -run "TestIsAvailable|TestGetBinaryPath" -v`
Expected: FAIL with "tr.IsAvailable undefined" or "tr.GetBinaryPath undefined"

**Step 3: Write minimal implementation**

Add to `transcriber.go` after `DeleteModel`:

```go
func (t *Transcriber) GetBinaryPath() string {
	if t.binPath != "" {
		return t.binPath
	}
	t.binPath = t.findWhisperBinary()
	return t.binPath
}

func (t *Transcriber) IsAvailable() bool {
	return t.GetBinaryPath() != ""
}
```

**Step 4: Run test to verify it passes**

Run: `cd /c/Users/nlafo/dev/ig2insights && go test ./internal/adapters/whisper/... -run "TestIsAvailable|TestGetBinaryPath" -v`
Expected: PASS

**Step 5: Commit**

```bash
cd /c/Users/nlafo/dev/ig2insights
git add internal/adapters/whisper/transcriber.go internal/adapters/whisper/transcriber_test.go
git commit -m "feat(whisper): add GetBinaryPath and IsAvailable methods"
```

---

## Task 3: Add getDownloadURL with platform support

**Files:**
- Modify: `internal/adapters/whisper/transcriber.go`
- Test: `internal/adapters/whisper/transcriber_test.go`

**Step 1: Write the failing test**

Add to `transcriber_test.go`:

```go
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
```

Add `"strings"` to test imports if not present.

**Step 2: Run test to verify it fails**

Run: `cd /c/Users/nlafo/dev/ig2insights && go test ./internal/adapters/whisper/... -run "TestGetDownloadURL|TestInstallationInstructions" -v`
Expected: FAIL with "tr.getDownloadURL undefined"

**Step 3: Write minimal implementation**

Add to `transcriber.go`:

```go
func (t *Transcriber) getDownloadURL() string {
	if runtime.GOOS == "windows" {
		return "https://github.com/ggerganov/whisper.cpp/releases/latest/download/whisper-bin-x64.zip"
	}
	return ""
}

func (t *Transcriber) InstallationInstructions() string {
	switch runtime.GOOS {
	case "windows":
		return "" // Auto-download available
	case "darwin":
		return "whisper.cpp not found. Install with:\n  brew install whisper-cpp"
	default:
		return "whisper.cpp not found. Build from source:\n  git clone https://github.com/ggerganov/whisper.cpp\n  cd whisper.cpp && make\n  sudo cp main /usr/local/bin/whisper"
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd /c/Users/nlafo/dev/ig2insights && go test ./internal/adapters/whisper/... -run "TestGetDownloadURL|TestInstallationInstructions" -v`
Expected: PASS

**Step 5: Commit**

```bash
cd /c/Users/nlafo/dev/ig2insights
git add internal/adapters/whisper/transcriber.go internal/adapters/whisper/transcriber_test.go
git commit -m "feat(whisper): add getDownloadURL and InstallationInstructions"
```

---

## Task 4: Add Install method with zip extraction

**Files:**
- Modify: `internal/adapters/whisper/transcriber.go`
- Test: `internal/adapters/whisper/transcriber_test.go`

**Step 1: Add archive/zip import**

Add `"archive/zip"` to imports in `transcriber.go`.

**Step 2: Write the failing test**

Add to `transcriber_test.go`:

```go
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

func TestExtractMainFromZip(t *testing.T) {
	tmpDir := t.TempDir()
	tr := NewTranscriber(tmpDir)

	// Create a test zip file with main.exe
	zipPath := filepath.Join(tmpDir, "test.zip")
	destPath := filepath.Join(tmpDir, "extracted.exe")

	// Create zip
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zipWriter := zip.NewWriter(zipFile)

	// Add main.exe to zip
	w, err := zipWriter.Create("main.exe")
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("fake binary content"))
	zipWriter.Close()
	zipFile.Close()

	// Extract
	err = tr.extractMainFromZip(zipPath, destPath)
	if err != nil {
		t.Fatalf("extractMainFromZip() failed: %v", err)
	}

	// Verify file exists
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(content) != "fake binary content" {
		t.Errorf("extracted content = %q, want 'fake binary content'", content)
	}
}

func TestExtractMainFromZip_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	tr := NewTranscriber(tmpDir)

	// Create a zip without main.exe
	zipPath := filepath.Join(tmpDir, "test.zip")
	destPath := filepath.Join(tmpDir, "extracted.exe")

	zipFile, _ := os.Create(zipPath)
	zipWriter := zip.NewWriter(zipFile)
	w, _ := zipWriter.Create("other.txt")
	w.Write([]byte("not the binary"))
	zipWriter.Close()
	zipFile.Close()

	err := tr.extractMainFromZip(zipPath, destPath)
	if err == nil {
		t.Error("extractMainFromZip() should fail when main.exe not in zip")
	}
}
```

Add `"archive/zip"` to test imports.

**Step 3: Run test to verify it fails**

Run: `cd /c/Users/nlafo/dev/ig2insights && go test ./internal/adapters/whisper/... -run "TestInstall|TestExtract" -v`
Expected: FAIL with undefined methods

**Step 4: Write minimal implementation**

Add to `transcriber.go`:

```go
func (t *Transcriber) Install(ctx context.Context, progress func(downloaded, total int64)) error {
	downloadURL := t.getDownloadURL()
	if downloadURL == "" {
		return fmt.Errorf("no prebuilt whisper.cpp binary for %s.\n%s", runtime.GOOS, t.InstallationInstructions())
	}

	binDir := config.BinDir()
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	// Download zip to temp file
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download whisper.cpp: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download whisper.cpp: HTTP %d", resp.StatusCode)
	}

	// Download to temp file
	tmpFile, err := os.CreateTemp("", "whisper-*.zip")
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

	// Extract main.exe from zip and rename to whisper.exe
	destPath := filepath.Join(binDir, whisperBinaryName())
	if err := t.extractMainFromZip(tmpPath, destPath); err != nil {
		return err
	}

	t.binPath = destPath
	return nil
}

func (t *Transcriber) extractMainFromZip(zipPath, destPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	// Look for main.exe in the zip
	var mainFile *zip.File
	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name == "main.exe" || name == "main" {
			mainFile = f
			break
		}
	}

	if mainFile == nil {
		return fmt.Errorf("main executable not found in whisper.cpp zip")
	}

	// Extract to destination
	src, err := mainFile.Open()
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

**Step 5: Run test to verify it passes**

Run: `cd /c/Users/nlafo/dev/ig2insights && go test ./internal/adapters/whisper/... -run "TestInstall|TestExtract" -v`
Expected: PASS

**Step 6: Commit**

```bash
cd /c/Users/nlafo/dev/ig2insights
git add internal/adapters/whisper/transcriber.go internal/adapters/whisper/transcriber_test.go
git commit -m "feat(whisper): add Install method with zip extraction"
```

---

## Task 5: Update deps command to show and install whisper

**Files:**
- Modify: `internal/adapters/cli/deps.go:14-15, 39-68, 92-119`

**Step 1: Update command description**

Change line 14-15:
```go
	cmd := &cobra.Command{
		Use:   "deps",
		Short: "Manage dependencies (yt-dlp, whisper.cpp)",
	}
```

**Step 2: Update runDepsStatus to show whisper status**

Replace `runDepsStatus` function:

```go
func runDepsStatus(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Dependency Status:")
	fmt.Println()

	// yt-dlp
	if app.Downloader.IsAvailable() {
		path := app.Downloader.GetBinaryPath()
		fmt.Printf("  yt-dlp:        installed (%s)\n", path)
	} else {
		fmt.Println("  yt-dlp:        not found")
	}

	// Whisper binary
	if app.Transcriber.IsAvailable() {
		path := app.Transcriber.GetBinaryPath()
		fmt.Printf("  whisper.cpp:   installed (%s)\n", path)
	} else {
		fmt.Println("  whisper.cpp:   not found")
	}

	// Whisper models
	models := app.Transcriber.AvailableModels()
	downloaded := 0
	for _, m := range models {
		if m.Downloaded {
			downloaded++
		}
	}
	fmt.Printf("  whisper models: %d/%d downloaded\n", downloaded, len(models))
	fmt.Println()

	return nil
}
```

**Step 3: Update runDepsInstall to install both**

Replace `runDepsInstall` function:

```go
func runDepsInstall(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	ctx := context.Background()
	progress := func(downloaded, total int64) {
		if total > 0 {
			pct := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rProgress: %.1f%%", pct)
		}
	}

	// Install yt-dlp
	if app.Downloader.IsAvailable() {
		fmt.Println("yt-dlp is already installed")
	} else {
		fmt.Println("Installing yt-dlp...")
		if err := app.Downloader.Install(ctx, progress); err != nil {
			return fmt.Errorf("failed to install yt-dlp: %w", err)
		}
		fmt.Println("\nyt-dlp installed")
	}

	// Install whisper.cpp
	if app.Transcriber.IsAvailable() {
		fmt.Println("whisper.cpp is already installed")
	} else {
		fmt.Println("Installing whisper.cpp...")
		if err := app.Transcriber.Install(ctx, progress); err != nil {
			return fmt.Errorf("failed to install whisper.cpp: %w", err)
		}
		fmt.Println("\nwhisper.cpp installed")
	}

	return nil
}
```

**Step 4: Run build to verify no errors**

Run: `cd /c/Users/nlafo/dev/ig2insights && go build ./...`
Expected: Success

**Step 5: Commit**

```bash
cd /c/Users/nlafo/dev/ig2insights
git add internal/adapters/cli/deps.go
git commit -m "feat(cli): update deps command to manage whisper.cpp"
```

---

## Task 6: Update root command to auto-install whisper

**Files:**
- Modify: `internal/adapters/cli/root.go:124-131`

**Step 1: Add whisper check after yt-dlp check**

After the yt-dlp installation block (around line 131), add:

```go
	if !app.Transcriber.IsAvailable() {
		instructions := app.Transcriber.InstallationInstructions()
		if instructions != "" {
			return fmt.Errorf(instructions)
		}
		fmt.Println("whisper.cpp not found. Installing...")
		if err := app.Transcriber.Install(context.Background(), printProgress); err != nil {
			return fmt.Errorf("failed to install whisper.cpp: %w", err)
		}
		fmt.Println("\n✓ whisper.cpp installed")
	}
```

**Step 2: Run build to verify no errors**

Run: `cd /c/Users/nlafo/dev/ig2insights && go build ./...`
Expected: Success

**Step 3: Commit**

```bash
cd /c/Users/nlafo/dev/ig2insights
git add internal/adapters/cli/root.go
git commit -m "feat(cli): auto-install whisper.cpp before transcribing"
```

---

## Task 7: Run all tests and final verification

**Step 1: Run all tests**

Run: `cd /c/Users/nlafo/dev/ig2insights && go test ./... -v`
Expected: All tests pass

**Step 2: Build the binary**

Run: `cd /c/Users/nlafo/dev/ig2insights && go build -o ig2insights.exe ./cmd/ig2insights`
Expected: Success

**Step 3: Test deps status command**

Run: `cd /c/Users/nlafo/dev/ig2insights && ./ig2insights.exe deps status`
Expected: Shows yt-dlp, whisper.cpp, and models status

**Step 4: Commit any final changes**

```bash
cd /c/Users/nlafo/dev/ig2insights
git status
# If any uncommitted changes, add and commit
```
