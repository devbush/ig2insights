package whisper

import (
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

// Model sizes in bytes (approximate)
var modelSizes = map[string]int64{
	"tiny":   75 * 1024 * 1024,
	"base":   140 * 1024 * 1024,
	"small":  462 * 1024 * 1024,
	"medium": 1500 * 1024 * 1024,
	"large":  3000 * 1024 * 1024,
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

func (t *Transcriber) AvailableModels() []ports.Model {
	models := []ports.Model{
		{Name: "tiny", Size: modelSizes["tiny"], Description: "~75MB, basic accuracy, very fast"},
		{Name: "base", Size: modelSizes["base"], Description: "~140MB, good accuracy, fast"},
		{Name: "small", Size: modelSizes["small"], Description: "~462MB, better accuracy, moderate speed"},
		{Name: "medium", Size: modelSizes["medium"], Description: "~1.5GB, great accuracy, slower"},
		{Name: "large", Size: modelSizes["large"], Description: "~3GB, best accuracy, slow"},
	}

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
	if _, ok := modelSizes[model]; !ok {
		return fmt.Errorf("unknown model: %s", model)
	}

	if err := os.MkdirAll(t.modelsDir, 0755); err != nil {
		return err
	}

	url := modelURL(model)
	destPath := t.modelPath(model)
	tempPath := destPath + ".tmp"

	// Use context-aware HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download model: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(tempPath)
	if err != nil {
		return err
	}

	// Track success to clean up partial downloads on failure
	success := false
	defer func() {
		out.Close()
		if !success {
			os.Remove(tempPath)
		}
	}()

	total := resp.ContentLength
	var downloaded int64

	buf := make([]byte, 32*1024)
	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
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
			return err
		}
	}

	out.Close()
	if err := os.Rename(tempPath, destPath); err != nil {
		return err
	}

	success = true
	return nil
}

func (t *Transcriber) DeleteModel(model string) error {
	return os.Remove(t.modelPath(model))
}

func (t *Transcriber) Transcribe(ctx context.Context, videoPath string, opts ports.TranscribeOpts) (*domain.Transcript, error) {
	model := opts.Model
	if model == "" {
		model = "small"
	}

	if !t.IsModelDownloaded(model) {
		return nil, domain.ErrModelNotFound
	}

	// Find whisper binary
	whisperBin := t.findWhisperBinary()
	if whisperBin == "" {
		return nil, fmt.Errorf("whisper binary not found (install whisper.cpp)")
	}

	// Create temp file for output
	tmpDir := os.TempDir()
	outputBase := filepath.Join(tmpDir, fmt.Sprintf("ig2insights_%d", time.Now().UnixNano()))

	args := []string{
		"-m", t.modelPath(model),
		"-f", videoPath,
		"-of", outputBase,
		"-oj", // JSON output
	}

	if opts.Language != "" {
		args = append(args, "-l", opts.Language)
	}

	cmd := exec.CommandContext(ctx, whisperBin, args...)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

	// Read JSON output
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

// Ensure Transcriber implements interface
var _ ports.Transcriber = (*Transcriber)(nil)
