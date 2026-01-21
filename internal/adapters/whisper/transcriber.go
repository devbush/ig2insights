package whisper

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/devbush/ig2insights/internal/config"
	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

// availableModels defines all supported Whisper models with their metadata
var availableModels = []ports.Model{
	{Name: "tiny", Size: 75 * 1024 * 1024, Description: "~75MB, basic accuracy, very fast"},
	{Name: "base", Size: 140 * 1024 * 1024, Description: "~140MB, good accuracy, fast"},
	{Name: "small", Size: 462 * 1024 * 1024, Description: "~462MB, better accuracy, moderate speed"},
	{Name: "medium", Size: 1500 * 1024 * 1024, Description: "~1.5GB, great accuracy, slower"},
	{Name: "large", Size: 3000 * 1024 * 1024, Description: "~3GB, best accuracy, slow"},
}

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

// NewTranscriber creates a new Whisper transcriber
func NewTranscriber(modelsDir string) *Transcriber {
	if modelsDir == "" {
		modelsDir = config.ModelsDir()
	}
	return &Transcriber{modelsDir: modelsDir}
}

func modelURL(name string) string {
	return fmt.Sprintf("https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-%s.bin", name)
}

func (t *Transcriber) modelPath(name string) string {
	return filepath.Join(t.modelsDir, fmt.Sprintf("ggml-%s.bin", name))
}

func isValidModel(name string) bool {
	for _, m := range availableModels {
		if m.Name == name {
			return true
		}
	}
	return false
}

// downloadWithProgress downloads from a URL to destPath with progress reporting and context cancellation.
// Partial downloads are cleaned up on failure.
func downloadWithProgress(ctx context.Context, url, destPath string, progress func(downloaded, total int64)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}

	downloadComplete := false
	defer func() {
		out.Close()
		if !downloadComplete {
			os.Remove(destPath)
		}
	}()

	total := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
			if progress != nil {
				progress(downloaded, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	downloadComplete = true
	return nil
}

func (t *Transcriber) AvailableModels() []ports.Model {
	models := make([]ports.Model, len(availableModels))
	copy(models, availableModels)

	for i := range models {
		models[i].Downloaded = t.IsModelDownloaded(models[i].Name)
	}

	return models
}

func (t *Transcriber) IsModelDownloaded(model string) bool {
	_, err := os.Stat(t.modelPath(model))
	return err == nil
}

func (t *Transcriber) DownloadModel(ctx context.Context, model string, progress func(downloaded, total int64)) error {
	if !isValidModel(model) {
		return fmt.Errorf("unknown model: %s", model)
	}

	if err := os.MkdirAll(t.modelsDir, 0755); err != nil {
		return err
	}

	destPath := t.modelPath(model)
	tempPath := destPath + ".tmp"

	if err := downloadWithProgress(ctx, modelURL(model), tempPath, progress); err != nil {
		return fmt.Errorf("failed to download model: %w", err)
	}

	if err := os.Rename(tempPath, destPath); err != nil {
		os.Remove(tempPath)
		return err
	}

	return nil
}

func (t *Transcriber) DeleteModel(model string) error {
	return os.Remove(t.modelPath(model))
}

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

func (t *Transcriber) Transcribe(ctx context.Context, videoPath string, opts ports.TranscribeOpts) (*domain.Transcript, error) {
	model := opts.Model
	if model == "" {
		model = "small"
	}

	if !t.IsModelDownloaded(model) {
		return nil, domain.ErrModelNotFound
	}

	whisperBin := t.GetBinaryPath()
	if whisperBin == "" {
		return nil, fmt.Errorf("whisper binary not found (install whisper.cpp)")
	}

	tmpDir := os.TempDir()
	outputBase := filepath.Join(tmpDir, fmt.Sprintf("ig2insights_%d", time.Now().UnixNano()))

	language := opts.Language
	if language == "" {
		language = "auto"
	}

	args := []string{
		"-m", t.modelPath(model),
		"-f", videoPath,
		"-of", outputBase,
		"-oj",
		"-l", language,
	}

	cmd := exec.CommandContext(ctx, whisperBin, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return nil, fmt.Errorf("failed to transcribe: %s", strings.TrimSpace(errMsg))
		}
		return nil, fmt.Errorf("failed to transcribe: %w", err)
	}

	jsonPath := outputBase + ".json"
	defer os.Remove(jsonPath)

	return t.parseWhisperJSON(jsonPath, model)
}

func (t *Transcriber) findWhisperBinary() string {
	names := []string{"whisper", "whisper-cpp", "main"}
	if runtime.GOOS == "windows" {
		names = []string{"whisper.exe", "whisper-cpp.exe", "main.exe"}
	}

	// Check bundled location
	for _, name := range names {
		bundled := filepath.Join(config.BinDir(), name)
		if _, err := os.Stat(bundled); err == nil {
			return bundled
		}
	}

	// Check PATH
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}

	return ""
}

func (t *Transcriber) parseWhisperJSON(path string, model string) (*domain.Transcript, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var output struct {
		Transcription []struct {
			Timestamps struct {
				From string `json:"from"`
				To   string `json:"to"`
			} `json:"timestamps"`
			Text string `json:"text"`
		} `json:"transcription"`
	}

	if err := json.Unmarshal(data, &output); err != nil {
		return nil, err
	}

	var segments []domain.Segment
	var fullText strings.Builder

	for _, item := range output.Transcription {
		start := parseTimestamp(item.Timestamps.From)
		end := parseTimestamp(item.Timestamps.To)
		text := strings.TrimSpace(item.Text)

		segments = append(segments, domain.Segment{
			Start: start,
			End:   end,
			Text:  text,
		})

		if fullText.Len() > 0 {
			fullText.WriteString(" ")
		}
		fullText.WriteString(text)
	}

	return &domain.Transcript{
		Text:          fullText.String(),
		Segments:      segments,
		Model:         model,
		Language:      "auto",
		TranscribedAt: time.Now(),
	}, nil
}

var timestampRegex = regexp.MustCompile(`(\d+):(\d+):(\d+)[,.](\d+)`)

func parseTimestamp(ts string) float64 {
	matches := timestampRegex.FindStringSubmatch(ts)
	if len(matches) != 5 {
		return 0
	}

	hours, _ := strconv.Atoi(matches[1])
	minutes, _ := strconv.Atoi(matches[2])
	seconds, _ := strconv.Atoi(matches[3])
	millis, _ := strconv.Atoi(matches[4])

	return float64(hours)*3600 + float64(minutes)*60 + float64(seconds) + float64(millis)/1000
}

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

func (t *Transcriber) Install(ctx context.Context, progress func(downloaded, total int64)) error {
	downloadURL := t.getDownloadURL()
	if downloadURL == "" {
		return fmt.Errorf("no prebuilt whisper.cpp binary for %s.\n%s", runtime.GOOS, t.InstallationInstructions())
	}

	binDir := config.BinDir()
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	tmpPath := filepath.Join(os.TempDir(), "whisper-download.zip")
	defer os.Remove(tmpPath)

	if err := downloadWithProgress(ctx, downloadURL, tmpPath, progress); err != nil {
		return fmt.Errorf("failed to download whisper.cpp: %w", err)
	}

	extractedFiles := []string{
		filepath.Join(binDir, whisperBinaryName()),
		filepath.Join(binDir, "ggml-base.dll"),
		filepath.Join(binDir, "ggml-cpu.dll"),
		filepath.Join(binDir, "ggml.dll"),
		filepath.Join(binDir, "whisper.dll"),
	}

	if err := t.extractWhisperFromZip(tmpPath, binDir); err != nil {
		for _, f := range extractedFiles {
			os.Remove(f)
		}
		return err
	}

	t.binPath = filepath.Join(binDir, whisperBinaryName())
	return nil
}

func (t *Transcriber) extractWhisperFromZip(zipPath, binDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	filesToExtract := map[string]string{
		"whisper-cli.exe": whisperBinaryName(),
		"ggml-base.dll":   "ggml-base.dll",
		"ggml-cpu.dll":    "ggml-cpu.dll",
		"ggml.dll":        "ggml.dll",
		"whisper.dll":     "whisper.dll",
	}

	extracted := make(map[string]bool)

	for _, f := range r.File {
		name := filepath.Base(f.Name)
		destName, needed := filesToExtract[name]
		if !needed {
			continue
		}

		destPath := filepath.Join(binDir, destName)
		if err := extractFile(f, destPath); err != nil {
			return fmt.Errorf("failed to extract %s: %w", name, err)
		}
		extracted[name] = true
	}

	if !extracted["whisper-cli.exe"] {
		return fmt.Errorf("whisper-cli.exe not found in whisper.cpp zip")
	}

	return nil
}

func extractFile(f *zip.File, destPath string) error {
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

	if runtime.GOOS != "windows" {
		if err := os.Chmod(destPath, 0755); err != nil {
			return err
		}
	}

	return nil
}

// Ensure Transcriber implements interface
var _ ports.Transcriber = (*Transcriber)(nil)
