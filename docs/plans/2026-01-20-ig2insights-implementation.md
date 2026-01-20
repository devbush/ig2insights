# ig2insights Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a CLI tool that transcribes Instagram Reels using yt-dlp and local Whisper.

**Architecture:** DDD with ports/adapters pattern. Domain layer has zero external dependencies. Application layer orchestrates use cases. Adapters implement ports for yt-dlp, whisper, cache, and CLI.

**Tech Stack:** Go 1.21+, Cobra (CLI), Bubbletea (TUI), whisper.cpp (transcription), yt-dlp (video download)

---

## Phase 1: Project Foundation

### Task 1.1: Initialize Go Module

**Files:**
- Create: `go.mod`
- Create: `cmd/ig2insights/main.go`

**Step 1: Initialize Go module**

Run:
```bash
cd ~/dev/ig2insights && go mod init github.com/devbush/ig2insights
```

Expected: `go.mod` created

**Step 2: Create minimal main.go**

Create `cmd/ig2insights/main.go`:
```go
package main

import "fmt"

func main() {
	fmt.Println("ig2insights")
}
```

**Step 3: Verify it compiles**

Run:
```bash
go build -o ig2insights ./cmd/ig2insights && ./ig2insights
```

Expected output: `ig2insights`

**Step 4: Commit**

```bash
git add go.mod cmd/
git commit -m "chore: initialize go module and main entry point"
```

---

### Task 1.2: Domain - Reel Entity

**Files:**
- Create: `internal/domain/reel.go`
- Create: `internal/domain/reel_test.go`

**Step 1: Write failing test for Reel parsing**

Create `internal/domain/reel_test.go`:
```go
package domain

import (
	"testing"
)

func TestParseReelInput_URL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantID   string
		wantErr  bool
	}{
		{
			name:    "full URL with /p/",
			input:   "https://www.instagram.com/p/DToLsd-EvGJ/",
			wantID:  "DToLsd-EvGJ",
			wantErr: false,
		},
		{
			name:    "full URL with /reel/",
			input:   "https://www.instagram.com/reel/DToLsd-EvGJ/",
			wantID:  "DToLsd-EvGJ",
			wantErr: false,
		},
		{
			name:    "URL without trailing slash",
			input:   "https://www.instagram.com/p/DToLsd-EvGJ",
			wantID:  "DToLsd-EvGJ",
			wantErr: false,
		},
		{
			name:    "just reel ID",
			input:   "DToLsd-EvGJ",
			wantID:  "DToLsd-EvGJ",
			wantErr: false,
		},
		{
			name:    "invalid input",
			input:   "",
			wantID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reel, err := ParseReelInput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseReelInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && reel.ID != tt.wantID {
				t.Errorf("ParseReelInput() ID = %v, want %v", reel.ID, tt.wantID)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/domain/... -v
```

Expected: FAIL - `ParseReelInput` not defined

**Step 3: Implement Reel entity**

Create `internal/domain/reel.go`:
```go
package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Reel represents an Instagram Reel
type Reel struct {
	ID              string
	URL             string
	Author          string
	Title           string
	DurationSeconds int
	ViewCount       int64
	FetchedAt       time.Time
}

// ReelURL builds the full Instagram URL for a reel
func (r *Reel) ReelURL() string {
	if r.URL != "" {
		return r.URL
	}
	return fmt.Sprintf("https://www.instagram.com/p/%s/", r.ID)
}

var (
	// Matches /p/ID or /reel/ID patterns
	reelURLPattern = regexp.MustCompile(`instagram\.com/(?:p|reel)/([A-Za-z0-9_-]+)`)
	// Valid reel ID pattern (alphanumeric, dash, underscore)
	reelIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

// ParseReelInput extracts a Reel from a URL or ID string
func ParseReelInput(input string) (*Reel, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	// Try to match URL pattern
	if matches := reelURLPattern.FindStringSubmatch(input); len(matches) > 1 {
		return &Reel{
			ID:  matches[1],
			URL: input,
		}, nil
	}

	// Check if it's a valid reel ID
	if reelIDPattern.MatchString(input) {
		return &Reel{
			ID: input,
		}, nil
	}

	return nil, fmt.Errorf("invalid reel URL or ID: %s", input)
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/domain/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/
git commit -m "feat(domain): add Reel entity with URL/ID parsing"
```

---

### Task 1.3: Domain - Transcript Entity

**Files:**
- Create: `internal/domain/transcript.go`
- Create: `internal/domain/transcript_test.go`

**Step 1: Write failing test for Transcript formatting**

Create `internal/domain/transcript_test.go`:
```go
package domain

import (
	"strings"
	"testing"
)

func TestTranscript_ToText(t *testing.T) {
	tr := &Transcript{
		Segments: []Segment{
			{Start: 0.0, End: 3.5, Text: "Hello world."},
			{Start: 3.5, End: 7.0, Text: "How are you?"},
		},
	}

	result := tr.ToText()
	expected := "Hello world. How are you?"

	if result != expected {
		t.Errorf("ToText() = %q, want %q", result, expected)
	}
}

func TestTranscript_ToSRT(t *testing.T) {
	tr := &Transcript{
		Segments: []Segment{
			{Start: 0.0, End: 3.5, Text: "Hello world."},
			{Start: 3.5, End: 7.2, Text: "How are you?"},
		},
	}

	result := tr.ToSRT()

	if !strings.Contains(result, "00:00:00,000 --> 00:00:03,500") {
		t.Errorf("ToSRT() missing first timestamp, got:\n%s", result)
	}
	if !strings.Contains(result, "Hello world.") {
		t.Errorf("ToSRT() missing first text")
	}
	if !strings.Contains(result, "00:00:03,500 --> 00:00:07,200") {
		t.Errorf("ToSRT() missing second timestamp, got:\n%s", result)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/domain/... -v
```

Expected: FAIL - `Transcript` not defined

**Step 3: Implement Transcript entity**

Create `internal/domain/transcript.go`:
```go
package domain

import (
	"fmt"
	"strings"
	"time"
)

// Segment represents a timed segment of transcribed text
type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// Transcript represents the full transcription result
type Transcript struct {
	Text          string    `json:"text"`
	Segments      []Segment `json:"segments"`
	Model         string    `json:"model"`
	Language      string    `json:"language"`
	TranscribedAt time.Time `json:"transcribed_at"`
}

// ToText returns plain text concatenation of all segments
func (t *Transcript) ToText() string {
	if t.Text != "" {
		return t.Text
	}

	var parts []string
	for _, seg := range t.Segments {
		parts = append(parts, strings.TrimSpace(seg.Text))
	}
	return strings.Join(parts, " ")
}

// ToSRT returns the transcript in SRT subtitle format
func (t *Transcript) ToSRT() string {
	var sb strings.Builder

	for i, seg := range t.Segments {
		// Sequence number
		sb.WriteString(fmt.Sprintf("%d\n", i+1))
		// Timestamps
		sb.WriteString(fmt.Sprintf("%s --> %s\n", formatSRTTime(seg.Start), formatSRTTime(seg.End)))
		// Text
		sb.WriteString(strings.TrimSpace(seg.Text))
		sb.WriteString("\n\n")
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// formatSRTTime converts seconds to SRT timestamp format (HH:MM:SS,mmm)
func formatSRTTime(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60
	millis := int((seconds - float64(int(seconds))) * 1000)

	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, millis)
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/domain/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/
git commit -m "feat(domain): add Transcript entity with text/SRT formatting"
```

---

### Task 1.4: Domain - Account Entity

**Files:**
- Create: `internal/domain/account.go`
- Create: `internal/domain/account_test.go`

**Step 1: Write failing test for Account parsing**

Create `internal/domain/account_test.go`:
```go
package domain

import "testing"

func TestParseAccountInput(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantUsername string
		wantErr      bool
	}{
		{
			name:         "full URL",
			input:        "https://www.instagram.com/npcfaizan/",
			wantUsername: "npcfaizan",
			wantErr:      false,
		},
		{
			name:         "URL without trailing slash",
			input:        "https://www.instagram.com/npcfaizan",
			wantUsername: "npcfaizan",
			wantErr:      false,
		},
		{
			name:         "with @ prefix",
			input:        "@npcfaizan",
			wantUsername: "npcfaizan",
			wantErr:      false,
		},
		{
			name:         "plain username",
			input:        "npcfaizan",
			wantUsername: "npcfaizan",
			wantErr:      false,
		},
		{
			name:         "empty input",
			input:        "",
			wantUsername: "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acc, err := ParseAccountInput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAccountInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && acc.Username != tt.wantUsername {
				t.Errorf("ParseAccountInput() Username = %v, want %v", acc.Username, tt.wantUsername)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/domain/... -v
```

Expected: FAIL - `ParseAccountInput` not defined

**Step 3: Implement Account entity**

Create `internal/domain/account.go`:
```go
package domain

import (
	"fmt"
	"regexp"
	"strings"
)

// SortOrder defines how to sort reels
type SortOrder string

const (
	SortLatest     SortOrder = "latest"
	SortMostViewed SortOrder = "most_viewed"
)

// Account represents an Instagram account
type Account struct {
	Username  string
	ReelCount int
}

// AccountURL builds the full Instagram URL for an account
func (a *Account) AccountURL() string {
	return fmt.Sprintf("https://www.instagram.com/%s/", a.Username)
}

var (
	// Matches instagram.com/username patterns (not /p/ or /reel/)
	accountURLPattern = regexp.MustCompile(`instagram\.com/([A-Za-z0-9_.]+)/?$`)
	// Valid username pattern
	usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.]+$`)
)

// ParseAccountInput extracts an Account from a URL or username string
func ParseAccountInput(input string) (*Account, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	// Remove @ prefix if present
	input = strings.TrimPrefix(input, "@")

	// Try to match URL pattern
	if matches := accountURLPattern.FindStringSubmatch(input); len(matches) > 1 {
		return &Account{
			Username: matches[1],
		}, nil
	}

	// Check if it's a valid username
	if usernamePattern.MatchString(input) {
		return &Account{
			Username: input,
		}, nil
	}

	return nil, fmt.Errorf("invalid account URL or username: %s", input)
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/domain/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/
git commit -m "feat(domain): add Account entity with URL/username parsing"
```

---

### Task 1.5: Domain - Errors

**Files:**
- Create: `internal/domain/errors.go`

**Step 1: Create domain errors**

Create `internal/domain/errors.go`:
```go
package domain

import "errors"

var (
	// ErrReelNotFound indicates the reel doesn't exist or is private
	ErrReelNotFound = errors.New("reel not found or is private")

	// ErrAccountNotFound indicates the account doesn't exist
	ErrAccountNotFound = errors.New("account not found")

	// ErrRateLimited indicates Instagram rate limiting
	ErrRateLimited = errors.New("rate limited by Instagram")

	// ErrNetworkFailure indicates a network error
	ErrNetworkFailure = errors.New("network failure")

	// ErrTranscriptionFailed indicates transcription error
	ErrTranscriptionFailed = errors.New("transcription failed")

	// ErrModelNotFound indicates the whisper model isn't downloaded
	ErrModelNotFound = errors.New("model not found")

	// ErrCacheExpired indicates cached item has expired
	ErrCacheExpired = errors.New("cache expired")

	// ErrCacheMiss indicates item not in cache
	ErrCacheMiss = errors.New("cache miss")
)
```

**Step 2: Commit**

```bash
git add internal/domain/errors.go
git commit -m "feat(domain): add domain error definitions"
```

---

### Task 1.6: Ports - Define Interfaces

**Files:**
- Create: `internal/ports/transcriber.go`
- Create: `internal/ports/downloader.go`
- Create: `internal/ports/fetcher.go`
- Create: `internal/ports/cache.go`

**Step 1: Create Transcriber interface**

Create `internal/ports/transcriber.go`:
```go
package ports

import (
	"context"

	"github.com/devbush/ig2insights/internal/domain"
)

// Model represents a Whisper model
type Model struct {
	Name        string
	Size        int64  // bytes
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
```

**Step 2: Create VideoDownloader interface**

Create `internal/ports/downloader.go`:
```go
package ports

import (
	"context"

	"github.com/devbush/ig2insights/internal/domain"
)

// DownloadResult contains the downloaded video info
type DownloadResult struct {
	VideoPath string
	Reel      *domain.Reel // Populated with metadata from download
}

// VideoDownloader handles video download from Instagram
type VideoDownloader interface {
	// Download fetches a video by reel ID, returns path to downloaded file
	Download(ctx context.Context, reelID string, destDir string) (*DownloadResult, error)

	// IsAvailable checks if the downloader is ready (yt-dlp installed)
	IsAvailable() bool

	// GetBinaryPath returns path to yt-dlp binary
	GetBinaryPath() string

	// Install downloads and installs yt-dlp
	Install(ctx context.Context, progress func(downloaded, total int64)) error

	// Update updates yt-dlp to latest version
	Update(ctx context.Context) error
}
```

**Step 3: Create AccountFetcher interface**

Create `internal/ports/fetcher.go`:
```go
package ports

import (
	"context"

	"github.com/devbush/ig2insights/internal/domain"
)

// AccountFetcher retrieves Instagram account information
type AccountFetcher interface {
	// GetAccount fetches account info including reel count
	GetAccount(ctx context.Context, username string) (*domain.Account, error)

	// ListReels fetches reels from an account
	ListReels(ctx context.Context, username string, sort domain.SortOrder, limit int) ([]*domain.Reel, error)
}
```

**Step 4: Create CacheStore interface**

Create `internal/ports/cache.go`:
```go
package ports

import (
	"context"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
)

// CachedItem represents a cached reel with transcript
type CachedItem struct {
	Reel       *domain.Reel
	Transcript *domain.Transcript
	VideoPath  string
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

// CacheStore handles persistent caching of reels and transcripts
type CacheStore interface {
	// Get retrieves a cached item by reel ID
	Get(ctx context.Context, reelID string) (*CachedItem, error)

	// Set stores an item in cache
	Set(ctx context.Context, reelID string, item *CachedItem) error

	// Delete removes a specific item from cache
	Delete(ctx context.Context, reelID string) error

	// CleanExpired removes all expired items
	CleanExpired(ctx context.Context) (int, error)

	// Clear removes all cached items
	Clear(ctx context.Context) error

	// GetCacheDir returns the cache directory path for a reel
	GetCacheDir(reelID string) string

	// Stats returns cache statistics
	Stats(ctx context.Context) (itemCount int, totalSize int64, err error)
}
```

**Step 5: Commit**

```bash
git add internal/ports/
git commit -m "feat(ports): add interface definitions for transcriber, downloader, fetcher, cache"
```

---

## Phase 2: Configuration & Utilities

### Task 2.1: Config Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write failing test for config loading**

Create `internal/config/config_test.go`:
```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Defaults.Model != "small" {
		t.Errorf("Default model = %s, want small", cfg.Defaults.Model)
	}
	if cfg.Defaults.Format != "text" {
		t.Errorf("Default format = %s, want text", cfg.Defaults.Format)
	}
	if cfg.Defaults.CacheTTL != "7d" {
		t.Errorf("Default cache TTL = %s, want 7d", cfg.Defaults.CacheTTL)
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		wantSecs int64
		wantErr  bool
	}{
		{"24h", 86400, false},
		{"7d", 604800, false},
		{"30d", 2592000, false},
		{"1h", 3600, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			dur, err := ParseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDuration(%s) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err == nil && int64(dur.Seconds()) != tt.wantSecs {
				t.Errorf("ParseDuration(%s) = %v, want %d seconds", tt.input, dur, tt.wantSecs)
			}
		})
	}
}

func TestConfig_Save_Load(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
	cfg.Defaults.Model = "large"

	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Defaults.Model != "large" {
		t.Errorf("Loaded model = %s, want large", loaded.Defaults.Model)
	}
}

func TestAppDir(t *testing.T) {
	dir := AppDir()
	if dir == "" {
		t.Error("AppDir() returned empty string")
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".ig2insights")
	if dir != expected {
		t.Errorf("AppDir() = %s, want %s", dir, expected)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/config/... -v
```

Expected: FAIL - package not found

**Step 3: Implement config package**

Create `internal/config/config.go`:
```go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Defaults DefaultsConfig `yaml:"defaults"`
	Paths    PathsConfig    `yaml:"paths"`
}

// DefaultsConfig holds default values
type DefaultsConfig struct {
	Model    string `yaml:"model"`
	Format   string `yaml:"format"`
	CacheTTL string `yaml:"cache_ttl"`
}

// PathsConfig holds custom path overrides
type PathsConfig struct {
	YtDlp string `yaml:"yt_dlp"`
}

// DefaultConfig returns configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Defaults: DefaultsConfig{
			Model:    "small",
			Format:   "text",
			CacheTTL: "7d",
		},
		Paths: PathsConfig{
			YtDlp: "",
		},
	}
}

// AppDir returns the application directory (~/.ig2insights)
func AppDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ig2insights"
	}
	return filepath.Join(home, ".ig2insights")
}

// ModelsDir returns the models directory
func ModelsDir() string {
	return filepath.Join(AppDir(), "models")
}

// CacheDir returns the cache directory
func CacheDir() string {
	return filepath.Join(AppDir(), "cache")
}

// BinDir returns the bin directory
func BinDir() string {
	return filepath.Join(AppDir(), "bin")
}

// ConfigPath returns the config file path
func ConfigPath() string {
	return filepath.Join(AppDir(), "config.yaml")
}

// EnsureDirs creates all required directories
func EnsureDirs() error {
	dirs := []string{AppDir(), ModelsDir(), CacheDir(), BinDir()}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// Load reads config from file, returns default if not exists
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}

// LoadDefault loads config from default path
func LoadDefault() (*Config, error) {
	return Load(ConfigPath())
}

// Save writes config to file
func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// SaveDefault saves config to default path
func (c *Config) SaveDefault() error {
	return c.Save(ConfigPath())
}

// GetCacheTTL returns the cache TTL as a duration
func (c *Config) GetCacheTTL() (time.Duration, error) {
	return ParseDuration(c.Defaults.CacheTTL)
}

var durationPattern = regexp.MustCompile(`^(\d+)(h|d)$`)

// ParseDuration parses duration strings like "24h", "7d", "30d"
func ParseDuration(s string) (time.Duration, error) {
	matches := durationPattern.FindStringSubmatch(s)
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid duration format: %s (use format like 24h, 7d)", s)
	}

	value, _ := strconv.Atoi(matches[1])
	unit := matches[2]

	switch unit {
	case "h":
		return time.Duration(value) * time.Hour, nil
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %s", unit)
	}
}
```

**Step 4: Add yaml dependency and run tests**

Run:
```bash
go get gopkg.in/yaml.v3
go test ./internal/config/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat(config): add configuration management with TTL parsing"
```

---

### Task 2.2: Cache Adapter

**Files:**
- Create: `internal/adapters/cache/store.go`
- Create: `internal/adapters/cache/store_test.go`

**Step 1: Write failing test for cache operations**

Create `internal/adapters/cache/store_test.go`:
```go
package cache

import (
	"context"
	"testing"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

func TestFileCache_SetGet(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewFileCache(tmpDir, 24*time.Hour)

	ctx := context.Background()
	item := &ports.CachedItem{
		Reel: &domain.Reel{
			ID:    "test123",
			Title: "Test Reel",
		},
		Transcript: &domain.Transcript{
			Text: "Hello world",
		},
		VideoPath: "/tmp/video.mp4",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Set
	err := cache.Set(ctx, "test123", item)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get
	got, err := cache.Get(ctx, "test123")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Reel.ID != "test123" {
		t.Errorf("Get() reel ID = %s, want test123", got.Reel.ID)
	}
	if got.Transcript.Text != "Hello world" {
		t.Errorf("Get() transcript text = %s, want Hello world", got.Transcript.Text)
	}
}

func TestFileCache_GetMiss(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewFileCache(tmpDir, 24*time.Hour)

	ctx := context.Background()
	_, err := cache.Get(ctx, "nonexistent")

	if err != domain.ErrCacheMiss {
		t.Errorf("Get() error = %v, want ErrCacheMiss", err)
	}
}

func TestFileCache_GetExpired(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewFileCache(tmpDir, 24*time.Hour)

	ctx := context.Background()
	item := &ports.CachedItem{
		Reel:      &domain.Reel{ID: "expired123"},
		CreatedAt: time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-24 * time.Hour), // Already expired
	}

	_ = cache.Set(ctx, "expired123", item)

	_, err := cache.Get(ctx, "expired123")
	if err != domain.ErrCacheExpired {
		t.Errorf("Get() error = %v, want ErrCacheExpired", err)
	}
}

func TestFileCache_CleanExpired(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewFileCache(tmpDir, 1*time.Millisecond)

	ctx := context.Background()

	// Add item that will expire immediately
	item := &ports.CachedItem{
		Reel:      &domain.Reel{ID: "willexpire"},
		CreatedAt: time.Now().Add(-1 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	}
	_ = cache.Set(ctx, "willexpire", item)

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Clean
	cleaned, err := cache.CleanExpired(ctx)
	if err != nil {
		t.Fatalf("CleanExpired() error = %v", err)
	}

	if cleaned != 1 {
		t.Errorf("CleanExpired() = %d, want 1", cleaned)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/adapters/cache/... -v
```

Expected: FAIL - package not found

**Step 3: Implement file cache**

Create `internal/adapters/cache/store.go`:
```go
package cache

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

// FileCache implements CacheStore using the filesystem
type FileCache struct {
	baseDir string
	ttl     time.Duration
}

// NewFileCache creates a new file-based cache
func NewFileCache(baseDir string, ttl time.Duration) *FileCache {
	return &FileCache{
		baseDir: baseDir,
		ttl:     ttl,
	}
}

// metaFile is the JSON structure stored in meta.json
type metaFile struct {
	Reel       *domain.Reel       `json:"reel"`
	Transcript *domain.Transcript `json:"transcript"`
	VideoPath  string             `json:"video_path"`
	CreatedAt  time.Time          `json:"created_at"`
	ExpiresAt  time.Time          `json:"expires_at"`
}

func (c *FileCache) GetCacheDir(reelID string) string {
	return filepath.Join(c.baseDir, reelID)
}

func (c *FileCache) metaPath(reelID string) string {
	return filepath.Join(c.GetCacheDir(reelID), "meta.json")
}

func (c *FileCache) Get(ctx context.Context, reelID string) (*ports.CachedItem, error) {
	metaPath := c.metaPath(reelID)

	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.ErrCacheMiss
		}
		return nil, err
	}

	var meta metaFile
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	// Check expiration
	if time.Now().After(meta.ExpiresAt) {
		return nil, domain.ErrCacheExpired
	}

	return &ports.CachedItem{
		Reel:       meta.Reel,
		Transcript: meta.Transcript,
		VideoPath:  meta.VideoPath,
		CreatedAt:  meta.CreatedAt,
		ExpiresAt:  meta.ExpiresAt,
	}, nil
}

func (c *FileCache) Set(ctx context.Context, reelID string, item *ports.CachedItem) error {
	cacheDir := c.GetCacheDir(reelID)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	meta := metaFile{
		Reel:       item.Reel,
		Transcript: item.Transcript,
		VideoPath:  item.VideoPath,
		CreatedAt:  item.CreatedAt,
		ExpiresAt:  item.ExpiresAt,
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.metaPath(reelID), data, 0644)
}

func (c *FileCache) Delete(ctx context.Context, reelID string) error {
	return os.RemoveAll(c.GetCacheDir(reelID))
}

func (c *FileCache) CleanExpired(ctx context.Context) (int, error) {
	entries, err := os.ReadDir(c.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	cleaned := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		reelID := entry.Name()
		_, err := c.Get(ctx, reelID)
		if err == domain.ErrCacheExpired {
			if err := c.Delete(ctx, reelID); err == nil {
				cleaned++
			}
		}
	}

	return cleaned, nil
}

func (c *FileCache) Clear(ctx context.Context) error {
	entries, err := os.ReadDir(c.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			_ = os.RemoveAll(filepath.Join(c.baseDir, entry.Name()))
		}
	}

	return nil
}

func (c *FileCache) Stats(ctx context.Context) (itemCount int, totalSize int64, err error) {
	entries, err := os.ReadDir(c.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		itemCount++

		// Calculate directory size
		dirPath := filepath.Join(c.baseDir, entry.Name())
		_ = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				totalSize += info.Size()
			}
			return nil
		})
	}

	return itemCount, totalSize, nil
}

// Ensure FileCache implements CacheStore
var _ ports.CacheStore = (*FileCache)(nil)
```

**Step 4: Run tests**

Run:
```bash
go test ./internal/adapters/cache/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/adapters/cache/
git commit -m "feat(cache): add file-based cache adapter with TTL support"
```

---

## Phase 3: Video Download (yt-dlp)

### Task 3.1: yt-dlp Adapter - Download

**Files:**
- Create: `internal/adapters/ytdlp/downloader.go`
- Create: `internal/adapters/ytdlp/downloader_test.go`

**Step 1: Write test for yt-dlp availability check**

Create `internal/adapters/ytdlp/downloader_test.go`:
```go
package ytdlp

import (
	"runtime"
	"testing"
)

func TestYtDlpBinaryName(t *testing.T) {
	name := binaryName()

	if runtime.GOOS == "windows" {
		if name != "yt-dlp.exe" {
			t.Errorf("binaryName() = %s, want yt-dlp.exe on Windows", name)
		}
	} else {
		if name != "yt-dlp" {
			t.Errorf("binaryName() = %s, want yt-dlp", name)
		}
	}
}

func TestBuildReelURL(t *testing.T) {
	url := buildReelURL("DToLsd-EvGJ")
	expected := "https://www.instagram.com/p/DToLsd-EvGJ/"

	if url != expected {
		t.Errorf("buildReelURL() = %s, want %s", url, expected)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/adapters/ytdlp/... -v
```

Expected: FAIL - package not found

**Step 3: Implement yt-dlp downloader**

Create `internal/adapters/ytdlp/downloader.go`:
```go
package ytdlp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/devbush/ig2insights/internal/config"
	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

// Downloader implements VideoDownloader and AccountFetcher using yt-dlp
type Downloader struct {
	binPath string
}

// NewDownloader creates a new yt-dlp downloader
func NewDownloader() *Downloader {
	return &Downloader{}
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "yt-dlp.exe"
	}
	return "yt-dlp"
}

func buildReelURL(reelID string) string {
	return fmt.Sprintf("https://www.instagram.com/p/%s/", reelID)
}

func (d *Downloader) findBinary() string {
	// Check bundled location first
	bundled := filepath.Join(config.BinDir(), binaryName())
	if _, err := os.Stat(bundled); err == nil {
		return bundled
	}

	// Check system PATH
	if path, err := exec.LookPath(binaryName()); err == nil {
		return path
	}

	return ""
}

func (d *Downloader) GetBinaryPath() string {
	if d.binPath != "" {
		return d.binPath
	}
	d.binPath = d.findBinary()
	return d.binPath
}

func (d *Downloader) IsAvailable() bool {
	return d.GetBinaryPath() != ""
}

func (d *Downloader) Download(ctx context.Context, reelID string, destDir string) (*ports.DownloadResult, error) {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return nil, fmt.Errorf("yt-dlp not found")
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	url := buildReelURL(reelID)
	outputTemplate := filepath.Join(destDir, "video.%(ext)s")

	// Run yt-dlp with JSON output for metadata
	args := []string{
		"--no-warnings",
		"--print-json",
		"-o", outputTemplate,
		url,
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "Private video") || strings.Contains(stderr, "Video unavailable") {
				return nil, domain.ErrReelNotFound
			}
			if strings.Contains(stderr, "rate") || strings.Contains(stderr, "429") {
				return nil, domain.ErrRateLimited
			}
			return nil, fmt.Errorf("yt-dlp failed: %s", stderr)
		}
		return nil, fmt.Errorf("yt-dlp failed: %w", err)
	}

	// Parse JSON output for metadata
	var info struct {
		ID          string  `json:"id"`
		Title       string  `json:"title"`
		Uploader    string  `json:"uploader"`
		Duration    float64 `json:"duration"`
		ViewCount   int64   `json:"view_count"`
		Ext         string  `json:"ext"`
		RequestedDownloads []struct {
			Filepath string `json:"filepath"`
		} `json:"requested_downloads"`
	}

	if err := json.Unmarshal(output, &info); err != nil {
		// Try to find the video file anyway
		matches, _ := filepath.Glob(filepath.Join(destDir, "video.*"))
		if len(matches) > 0 {
			return &ports.DownloadResult{
				VideoPath: matches[0],
				Reel: &domain.Reel{
					ID:        reelID,
					FetchedAt: time.Now(),
				},
			}, nil
		}
		return nil, fmt.Errorf("failed to parse yt-dlp output: %w", err)
	}

	videoPath := filepath.Join(destDir, fmt.Sprintf("video.%s", info.Ext))
	if len(info.RequestedDownloads) > 0 {
		videoPath = info.RequestedDownloads[0].Filepath
	}

	return &ports.DownloadResult{
		VideoPath: videoPath,
		Reel: &domain.Reel{
			ID:              reelID,
			URL:             url,
			Author:          info.Uploader,
			Title:           info.Title,
			DurationSeconds: int(info.Duration),
			ViewCount:       info.ViewCount,
			FetchedAt:       time.Now(),
		},
	}, nil
}

func (d *Downloader) Install(ctx context.Context, progress func(downloaded, total int64)) error {
	binDir := config.BinDir()
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	downloadURL := d.getDownloadURL()
	destPath := filepath.Join(binDir, binaryName())

	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download yt-dlp: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download yt-dlp: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	var downloaded int64

	buf := make([]byte, 32*1024)
	for {
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

	// Make executable on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(destPath, 0755); err != nil {
			return err
		}
	}

	d.binPath = destPath
	return nil
}

func (d *Downloader) getDownloadURL() string {
	base := "https://github.com/yt-dlp/yt-dlp/releases/latest/download/"

	switch runtime.GOOS {
	case "windows":
		return base + "yt-dlp.exe"
	case "darwin":
		return base + "yt-dlp_macos"
	default:
		return base + "yt-dlp"
	}
}

func (d *Downloader) Update(ctx context.Context) error {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return fmt.Errorf("yt-dlp not installed")
	}

	cmd := exec.CommandContext(ctx, binPath, "-U")
	return cmd.Run()
}

// Ensure Downloader implements interfaces
var _ ports.VideoDownloader = (*Downloader)(nil)
```

**Step 4: Run tests**

Run:
```bash
go test ./internal/adapters/ytdlp/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/adapters/ytdlp/
git commit -m "feat(ytdlp): add video downloader adapter with auto-install"
```

---

### Task 3.2: yt-dlp Adapter - Account Fetcher

**Files:**
- Modify: `internal/adapters/ytdlp/downloader.go`
- Update: `internal/adapters/ytdlp/downloader_test.go`

**Step 1: Add AccountFetcher implementation to downloader.go**

Add to `internal/adapters/ytdlp/downloader.go`:
```go
func (d *Downloader) GetAccount(ctx context.Context, username string) (*domain.Account, error) {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return nil, fmt.Errorf("yt-dlp not found")
	}

	// Use playlist extraction to get account info
	url := fmt.Sprintf("https://www.instagram.com/%s/reels/", username)

	args := []string{
		"--no-warnings",
		"--flat-playlist",
		"--print-json",
		"-I", "1:1", // Only fetch first item to get playlist info
		url,
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "not found") || strings.Contains(stderr, "404") {
				return nil, domain.ErrAccountNotFound
			}
		}
		return nil, fmt.Errorf("failed to fetch account: %w", err)
	}

	// Count entries to estimate reel count
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	return &domain.Account{
		Username:  username,
		ReelCount: len(lines), // Approximate from flat playlist
	}, nil
}

func (d *Downloader) ListReels(ctx context.Context, username string, sort domain.SortOrder, limit int) ([]*domain.Reel, error) {
	binPath := d.GetBinaryPath()
	if binPath == "" {
		return nil, fmt.Errorf("yt-dlp not found")
	}

	url := fmt.Sprintf("https://www.instagram.com/%s/reels/", username)

	args := []string{
		"--no-warnings",
		"--flat-playlist",
		"--print-json",
		"-I", fmt.Sprintf("1:%d", limit),
		url,
	}

	// Note: yt-dlp returns chronological order by default
	// For "most viewed", we'd need to fetch all and sort client-side
	// This is a limitation - Instagram doesn't expose a "top" endpoint easily

	cmd := exec.CommandContext(ctx, binPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list reels: %w", err)
	}

	var reels []*domain.Reel
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		var info struct {
			ID        string  `json:"id"`
			Title     string  `json:"title"`
			Uploader  string  `json:"uploader"`
			Duration  float64 `json:"duration"`
			ViewCount int64   `json:"view_count"`
		}

		if err := json.Unmarshal([]byte(line), &info); err != nil {
			continue
		}

		reels = append(reels, &domain.Reel{
			ID:              info.ID,
			Author:          info.Uploader,
			Title:           info.Title,
			DurationSeconds: int(info.Duration),
			ViewCount:       info.ViewCount,
			FetchedAt:       time.Now(),
		})
	}

	// Sort by view count if requested
	if sort == domain.SortMostViewed {
		sortByViews(reels)
	}

	return reels, nil
}

func sortByViews(reels []*domain.Reel) {
	for i := 0; i < len(reels)-1; i++ {
		for j := i + 1; j < len(reels); j++ {
			if reels[j].ViewCount > reels[i].ViewCount {
				reels[i], reels[j] = reels[j], reels[i]
			}
		}
	}
}

// Ensure Downloader implements AccountFetcher
var _ ports.AccountFetcher = (*Downloader)(nil)
```

**Step 2: Commit**

```bash
git add internal/adapters/ytdlp/
git commit -m "feat(ytdlp): add account fetcher for listing reels"
```

---

## Phase 4: Transcription (Whisper)

### Task 4.1: Whisper Adapter - Shell Implementation

**Files:**
- Create: `internal/adapters/whisper/transcriber.go`
- Create: `internal/adapters/whisper/transcriber_test.go`

Note: We'll shell out to whisper.cpp CLI for simplicity. CGO bindings can be added later.

**Step 1: Write test for model info**

Create `internal/adapters/whisper/transcriber_test.go`:
```go
package whisper

import (
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
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/adapters/whisper/... -v
```

Expected: FAIL - package not found

**Step 3: Implement Whisper transcriber**

Create `internal/adapters/whisper/transcriber.go`:
```go
package whisper

import (
	"bufio"
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

	resp, err := http.Get(url)
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
	defer out.Close()

	total := resp.ContentLength
	var downloaded int64

	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				os.Remove(tempPath)
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
			os.Remove(tempPath)
			return err
		}
	}

	out.Close()
	return os.Rename(tempPath, destPath)
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
```

**Step 4: Run tests**

Run:
```bash
go test ./internal/adapters/whisper/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/adapters/whisper/
git commit -m "feat(whisper): add transcriber adapter with model management"
```

---

## Phase 5: Application Layer

### Task 5.1: TranscribeReel Use Case

**Files:**
- Create: `internal/application/transcribe.go`
- Create: `internal/application/transcribe_test.go`

**Step 1: Write test with mocks**

Create `internal/application/transcribe_test.go`:
```go
package application

import (
	"context"
	"testing"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

// Mock implementations for testing
type mockCache struct {
	items map[string]*ports.CachedItem
}

func newMockCache() *mockCache {
	return &mockCache{items: make(map[string]*ports.CachedItem)}
}

func (m *mockCache) Get(ctx context.Context, reelID string) (*ports.CachedItem, error) {
	if item, ok := m.items[reelID]; ok {
		return item, nil
	}
	return nil, domain.ErrCacheMiss
}

func (m *mockCache) Set(ctx context.Context, reelID string, item *ports.CachedItem) error {
	m.items[reelID] = item
	return nil
}

func (m *mockCache) Delete(ctx context.Context, reelID string) error {
	delete(m.items, reelID)
	return nil
}

func (m *mockCache) CleanExpired(ctx context.Context) (int, error) { return 0, nil }
func (m *mockCache) Clear(ctx context.Context) error              { return nil }
func (m *mockCache) GetCacheDir(reelID string) string             { return "/tmp/" + reelID }
func (m *mockCache) Stats(ctx context.Context) (int, int64, error) {
	return len(m.items), 0, nil
}

type mockDownloader struct {
	available bool
}

func (m *mockDownloader) Download(ctx context.Context, reelID string, destDir string) (*ports.DownloadResult, error) {
	return &ports.DownloadResult{
		VideoPath: destDir + "/video.mp4",
		Reel: &domain.Reel{
			ID:        reelID,
			Title:     "Test Reel",
			Author:    "testuser",
			FetchedAt: time.Now(),
		},
	}, nil
}

func (m *mockDownloader) IsAvailable() bool                                               { return m.available }
func (m *mockDownloader) GetBinaryPath() string                                           { return "/usr/bin/yt-dlp" }
func (m *mockDownloader) Install(ctx context.Context, progress func(int64, int64)) error  { return nil }
func (m *mockDownloader) Update(ctx context.Context) error                                { return nil }

type mockTranscriber struct {
	modelDownloaded bool
}

func (m *mockTranscriber) Transcribe(ctx context.Context, videoPath string, opts ports.TranscribeOpts) (*domain.Transcript, error) {
	return &domain.Transcript{
		Text: "Hello world transcription",
		Segments: []domain.Segment{
			{Start: 0, End: 3.5, Text: "Hello world transcription"},
		},
		Model:         opts.Model,
		TranscribedAt: time.Now(),
	}, nil
}

func (m *mockTranscriber) AvailableModels() []ports.Model {
	return []ports.Model{{Name: "small", Size: 462 * 1024 * 1024, Downloaded: m.modelDownloaded}}
}

func (m *mockTranscriber) IsModelDownloaded(model string) bool { return m.modelDownloaded }
func (m *mockTranscriber) DownloadModel(ctx context.Context, model string, progress func(int64, int64)) error {
	return nil
}
func (m *mockTranscriber) DeleteModel(model string) error { return nil }

func TestTranscribeService_Transcribe(t *testing.T) {
	cache := newMockCache()
	downloader := &mockDownloader{available: true}
	transcriber := &mockTranscriber{modelDownloaded: true}

	svc := NewTranscribeService(cache, downloader, transcriber, 24*time.Hour)

	ctx := context.Background()
	result, err := svc.Transcribe(ctx, "test123", TranscribeOptions{
		Model:   "small",
		NoCache: false,
	})

	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}

	if result.Transcript.Text != "Hello world transcription" {
		t.Errorf("Transcript text = %s, want 'Hello world transcription'", result.Transcript.Text)
	}

	// Verify it was cached
	cached, err := cache.Get(ctx, "test123")
	if err != nil {
		t.Errorf("Item should be cached, got error: %v", err)
	}
	if cached.Transcript.Text != "Hello world transcription" {
		t.Errorf("Cached transcript text mismatch")
	}
}

func TestTranscribeService_CacheHit(t *testing.T) {
	cache := newMockCache()
	downloader := &mockDownloader{available: true}
	transcriber := &mockTranscriber{modelDownloaded: true}

	// Pre-populate cache
	cache.Set(context.Background(), "cached123", &ports.CachedItem{
		Reel: &domain.Reel{ID: "cached123"},
		Transcript: &domain.Transcript{
			Text: "Cached result",
		},
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	svc := NewTranscribeService(cache, downloader, transcriber, 24*time.Hour)

	ctx := context.Background()
	result, err := svc.Transcribe(ctx, "cached123", TranscribeOptions{})

	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}

	if result.Transcript.Text != "Cached result" {
		t.Errorf("Should return cached result, got: %s", result.Transcript.Text)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/application/... -v
```

Expected: FAIL - package not found

**Step 3: Implement TranscribeService**

Create `internal/application/transcribe.go`:
```go
package application

import (
	"context"
	"time"

	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

// TranscribeOptions configures the transcription
type TranscribeOptions struct {
	Model   string
	Format  string // text, srt, json
	NoCache bool
}

// TranscribeResult contains the transcription result
type TranscribeResult struct {
	Reel       *domain.Reel
	Transcript *domain.Transcript
	FromCache  bool
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

// Transcribe processes a reel and returns its transcript
func (s *TranscribeService) Transcribe(ctx context.Context, reelID string, opts TranscribeOptions) (*TranscribeResult, error) {
	// Check cache first (unless bypassed)
	if !opts.NoCache {
		cached, err := s.cache.Get(ctx, reelID)
		if err == nil {
			return &TranscribeResult{
				Reel:       cached.Reel,
				Transcript: cached.Transcript,
				FromCache:  true,
			}, nil
		}
	}

	// Download video
	cacheDir := s.cache.GetCacheDir(reelID)
	downloadResult, err := s.downloader.Download(ctx, reelID, cacheDir)
	if err != nil {
		return nil, err
	}

	// Transcribe
	model := opts.Model
	if model == "" {
		model = "small"
	}

	transcript, err := s.transcriber.Transcribe(ctx, downloadResult.VideoPath, ports.TranscribeOpts{
		Model: model,
	})
	if err != nil {
		return nil, err
	}

	// Cache result
	now := time.Now()
	cacheItem := &ports.CachedItem{
		Reel:       downloadResult.Reel,
		Transcript: transcript,
		VideoPath:  downloadResult.VideoPath,
		CreatedAt:  now,
		ExpiresAt:  now.Add(s.cacheTTL),
	}

	if err := s.cache.Set(ctx, reelID, cacheItem); err != nil {
		// Log but don't fail on cache error
	}

	return &TranscribeResult{
		Reel:       downloadResult.Reel,
		Transcript: transcript,
		FromCache:  false,
	}, nil
}
```

**Step 4: Run tests**

Run:
```bash
go test ./internal/application/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/application/
git commit -m "feat(application): add TranscribeService use case"
```

---

### Task 5.2: BrowseAccount Use Case

**Files:**
- Create: `internal/application/browse.go`

**Step 1: Implement BrowseService**

Create `internal/application/browse.go`:
```go
package application

import (
	"context"

	"github.com/devbush/ig2insights/internal/domain"
	"github.com/devbush/ig2insights/internal/ports"
)

// BrowseService handles account browsing operations
type BrowseService struct {
	fetcher ports.AccountFetcher
}

// NewBrowseService creates a new browse service
func NewBrowseService(fetcher ports.AccountFetcher) *BrowseService {
	return &BrowseService{fetcher: fetcher}
}

// GetAccount retrieves account information
func (s *BrowseService) GetAccount(ctx context.Context, username string) (*domain.Account, error) {
	return s.fetcher.GetAccount(ctx, username)
}

// ListReels retrieves reels from an account
func (s *BrowseService) ListReels(ctx context.Context, username string, sort domain.SortOrder, limit int) ([]*domain.Reel, error) {
	return s.fetcher.ListReels(ctx, username, sort, limit)
}
```

**Step 2: Commit**

```bash
git add internal/application/browse.go
git commit -m "feat(application): add BrowseService use case"
```

---

### Task 5.3: CacheManagement Use Case

**Files:**
- Create: `internal/application/cache.go`

**Step 1: Implement CacheService**

Create `internal/application/cache.go`:
```go
package application

import (
	"context"

	"github.com/devbush/ig2insights/internal/ports"
)

// CacheStats holds cache statistics
type CacheStats struct {
	ItemCount int
	TotalSize int64
}

// CacheService handles cache management operations
type CacheService struct {
	cache ports.CacheStore
}

// NewCacheService creates a new cache service
func NewCacheService(cache ports.CacheStore) *CacheService {
	return &CacheService{cache: cache}
}

// Stats returns cache statistics
func (s *CacheService) Stats(ctx context.Context) (*CacheStats, error) {
	count, size, err := s.cache.Stats(ctx)
	if err != nil {
		return nil, err
	}
	return &CacheStats{
		ItemCount: count,
		TotalSize: size,
	}, nil
}

// CleanExpired removes expired cache entries
func (s *CacheService) CleanExpired(ctx context.Context) (int, error) {
	return s.cache.CleanExpired(ctx)
}

// Clear removes all cache entries
func (s *CacheService) Clear(ctx context.Context) error {
	return s.cache.Clear(ctx)
}
```

**Step 2: Commit**

```bash
git add internal/application/cache.go
git commit -m "feat(application): add CacheService use case"
```

---

## Phase 6: CLI Implementation

### Task 6.1: CLI Setup with Cobra

**Files:**
- Create: `internal/adapters/cli/root.go`
- Modify: `cmd/ig2insights/main.go`

**Step 1: Add Cobra dependency**

Run:
```bash
go get github.com/spf13/cobra
```

**Step 2: Create root command**

Create `internal/adapters/cli/root.go`:
```go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	formatFlag   string
	modelFlag    string
	cacheTTLFlag string
	noCacheFlag  bool
	outputFlag   string
	quietFlag    bool
)

// NewRootCmd creates the root command
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ig2insights [reel-url|reel-id]",
		Short: "Transcribe Instagram Reels",
		Long: `ig2insights is a CLI tool that transcribes Instagram Reels.

Provide a reel URL or ID to transcribe it, or run without arguments
for an interactive menu.`,
		RunE: runRoot,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&formatFlag, "format", "", "Output format: text, srt, json")
	rootCmd.PersistentFlags().StringVar(&modelFlag, "model", "small", "Whisper model: tiny, base, small, medium, large")
	rootCmd.PersistentFlags().StringVar(&cacheTTLFlag, "cache-ttl", "7d", "Cache lifetime (e.g., 24h, 7d)")
	rootCmd.PersistentFlags().BoolVar(&noCacheFlag, "no-cache", false, "Skip cache")
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "", "Output file path")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress progress output")

	// Add subcommands
	rootCmd.AddCommand(NewAccountCmd())
	rootCmd.AddCommand(NewCacheCmd())
	rootCmd.AddCommand(NewModelCmd())
	rootCmd.AddCommand(NewDepsCmd())

	return rootCmd
}

func runRoot(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// No arguments - show interactive menu
		return runInteractiveMenu()
	}

	// Transcribe the provided reel
	return runTranscribe(args[0])
}

func runInteractiveMenu() error {
	// TODO: Implement with bubbletea
	fmt.Println("Interactive menu not yet implemented")
	fmt.Println("Usage: ig2insights <reel-url|reel-id>")
	return nil
}

func runTranscribe(input string) error {
	// TODO: Implement transcription flow
	fmt.Printf("Transcribing: %s\n", input)
	return nil
}

// Execute runs the CLI
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
```

**Step 3: Update main.go**

Replace `cmd/ig2insights/main.go`:
```go
package main

import (
	"github.com/devbush/ig2insights/internal/adapters/cli"
)

func main() {
	cli.Execute()
}
```

**Step 4: Create placeholder subcommands**

Create `internal/adapters/cli/account.go`:
```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	latestFlag int
	topFlag    int
)

// NewAccountCmd creates the account subcommand
func NewAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account <username|url>",
		Short: "Browse and transcribe reels from an account",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runAccount,
	}

	cmd.Flags().IntVar(&latestFlag, "latest", 0, "Transcribe N most recent reels")
	cmd.Flags().IntVar(&topFlag, "top", 0, "Transcribe N most viewed reels")

	return cmd
}

func runAccount(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// Interactive: prompt for username
		fmt.Println("Account browsing not yet implemented")
		return nil
	}

	username := args[0]
	fmt.Printf("Browsing account: %s\n", username)

	if latestFlag > 0 {
		fmt.Printf("Fetching %d latest reels\n", latestFlag)
	} else if topFlag > 0 {
		fmt.Printf("Fetching %d top reels\n", topFlag)
	} else {
		// Interactive: show scrollable list
		fmt.Println("Interactive reel selection not yet implemented")
	}

	return nil
}
```

Create `internal/adapters/cli/cache.go`:
```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var clearAllFlag bool

// NewCacheCmd creates the cache subcommand
func NewCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage cached transcripts",
	}

	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear cache entries",
		RunE:  runCacheClear,
	}
	clearCmd.Flags().BoolVar(&clearAllFlag, "all", false, "Clear all cache entries")

	cmd.AddCommand(clearCmd)

	return cmd
}

func runCacheClear(cmd *cobra.Command, args []string) error {
	if clearAllFlag {
		fmt.Println("Clearing all cache...")
	} else {
		fmt.Println("Clearing expired cache entries...")
	}
	return nil
}
```

Create `internal/adapters/cli/model.go`:
```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewModelCmd creates the model subcommand
func NewModelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Manage Whisper models",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available models",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Model list not yet implemented")
			return nil
		},
	}

	downloadCmd := &cobra.Command{
		Use:   "download <model>",
		Short: "Download a model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Downloading model: %s\n", args[0])
			return nil
		},
	}

	removeCmd := &cobra.Command{
		Use:   "remove <model>",
		Short: "Remove a downloaded model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Removing model: %s\n", args[0])
			return nil
		},
	}

	cmd.AddCommand(listCmd, downloadCmd, removeCmd)
	return cmd
}
```

Create `internal/adapters/cli/deps.go`:
```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewDepsCmd creates the deps subcommand
func NewDepsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deps",
		Short: "Manage dependencies (yt-dlp)",
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show dependency status",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Dependency status not yet implemented")
			return nil
		},
	}

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update yt-dlp",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Updating yt-dlp...")
			return nil
		},
	}

	cmd.AddCommand(statusCmd, updateCmd)
	return cmd
}
```

**Step 5: Build and test**

Run:
```bash
go build -o ig2insights ./cmd/ig2insights && ./ig2insights --help
```

Expected: Help output showing all commands

**Step 6: Commit**

```bash
git add cmd/ internal/adapters/cli/ go.mod go.sum
git commit -m "feat(cli): add Cobra-based CLI with subcommands"
```

---

### Task 6.2: Wire Up Dependencies

**Files:**
- Create: `internal/adapters/cli/app.go`
- Modify: `internal/adapters/cli/root.go`

**Step 1: Create app container**

Create `internal/adapters/cli/app.go`:
```go
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
```

**Step 2: Commit**

```bash
git add internal/adapters/cli/app.go
git commit -m "feat(cli): add dependency injection container"
```

---

### Task 6.3: Implement Transcribe Command

**Files:**
- Modify: `internal/adapters/cli/root.go`

**Step 1: Implement runTranscribe**

Update `runTranscribe` in `internal/adapters/cli/root.go`:
```go
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/devbush/ig2insights/internal/application"
	"github.com/devbush/ig2insights/internal/config"
	"github.com/devbush/ig2insights/internal/domain"
	"github.com/spf13/cobra"
)

// ... (keep existing code, update runTranscribe)

func runTranscribe(input string) error {
	app, err := GetApp()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Parse input
	reel, err := domain.ParseReelInput(input)
	if err != nil {
		return err
	}

	// Check dependencies
	if !app.Downloader.IsAvailable() {
		fmt.Println("yt-dlp not found. Installing...")
		if err := app.Downloader.Install(context.Background(), printProgress); err != nil {
			return fmt.Errorf("failed to install yt-dlp: %w", err)
		}
		fmt.Println("\n✓ yt-dlp installed")
	}

	model := modelFlag
	if model == "" {
		model = app.Config.Defaults.Model
	}

	if !app.Transcriber.IsModelDownloaded(model) {
		fmt.Printf("Model '%s' not found. Downloading...\n", model)
		if err := app.Transcriber.DownloadModel(context.Background(), model, printProgress); err != nil {
			return fmt.Errorf("failed to download model: %w", err)
		}
		fmt.Println("\n✓ Model downloaded")
	}

	// Transcribe
	if !quietFlag {
		fmt.Printf("Transcribing %s...\n", reel.ID)
	}

	ctx := context.Background()
	result, err := app.TranscribeSvc.Transcribe(ctx, reel.ID, application.TranscribeOptions{
		Model:   model,
		NoCache: noCacheFlag,
	})
	if err != nil {
		return err
	}

	if result.FromCache && !quietFlag {
		fmt.Println("(from cache)")
	}

	// Output
	return outputResult(result)
}

func printProgress(downloaded, total int64) {
	if quietFlag {
		return
	}
	if total > 0 {
		pct := float64(downloaded) / float64(total) * 100
		fmt.Printf("\rDownloading... %.1f%%", pct)
	}
}

func outputResult(result *application.TranscribeResult) error {
	format := formatFlag
	if format == "" {
		format = "text" // TODO: prompt user if interactive
	}

	var output string
	switch format {
	case "text":
		output = result.Transcript.ToText()
	case "srt":
		output = result.Transcript.ToSRT()
	case "json":
		data := map[string]interface{}{
			"reel":       result.Reel,
			"transcript": result.Transcript,
		}
		jsonBytes, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		output = string(jsonBytes)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}

	if outputFlag != "" {
		return os.WriteFile(outputFlag, []byte(output), 0644)
	}

	fmt.Println(output)
	return nil
}
```

**Step 2: Update imports and commit**

Run:
```bash
go build ./cmd/ig2insights
git add internal/adapters/cli/
git commit -m "feat(cli): implement transcribe command with progress output"
```

---

### Task 6.4: Add Bubbletea for Interactive UI

**Files:**
- Create: `internal/adapters/cli/tui/menu.go`
- Create: `internal/adapters/cli/tui/list.go`

**Step 1: Add bubbletea dependencies**

Run:
```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss
```

**Step 2: Create interactive menu**

Create `internal/adapters/cli/tui/menu.go`:
```go
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

// MenuOption represents a menu choice
type MenuOption struct {
	Label string
	Value string
}

// MenuModel is the bubbletea model for the main menu
type MenuModel struct {
	options  []MenuOption
	cursor   int
	selected string
}

// NewMenuModel creates a new menu
func NewMenuModel(options []MenuOption) MenuModel {
	return MenuModel{
		options: options,
	}
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.options[m.cursor].Value
			return m, tea.Quit
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	s := "? What would you like to do?\n\n"

	for i, opt := range m.options {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}
		s += fmt.Sprintf("%s%s\n", cursor, style.Render(opt.Label))
	}

	s += "\n(↑/↓ to navigate, enter to select, q to quit)\n"
	return s
}

// Selected returns the selected value
func (m MenuModel) Selected() string {
	return m.selected
}

// RunMenu displays the menu and returns the selection
func RunMenu(options []MenuOption) (string, error) {
	model := NewMenuModel(options)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	return finalModel.(MenuModel).Selected(), nil
}
```

**Step 3: Create reel list with checkboxes**

Create `internal/adapters/cli/tui/list.go`:
```go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/devbush/ig2insights/internal/domain"
)

var (
	checkedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	uncheckedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	titleStyle     = lipgloss.NewStyle().Bold(true)
)

// ReelListModel is the bubbletea model for reel selection
type ReelListModel struct {
	reels    []*domain.Reel
	cursor   int
	selected map[int]bool
	done     bool
}

// NewReelListModel creates a new reel list
func NewReelListModel(reels []*domain.Reel) ReelListModel {
	return ReelListModel{
		reels:    reels,
		selected: make(map[int]bool),
	}
}

func (m ReelListModel) Init() tea.Cmd {
	return nil
}

func (m ReelListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.reels)-1 {
				m.cursor++
			}
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "enter":
			m.done = true
			return m, tea.Quit
		case "q", "ctrl+c":
			m.selected = make(map[int]bool) // Clear selection
			return m, tea.Quit
		case "a":
			// Select all
			for i := range m.reels {
				m.selected[i] = true
			}
		case "n":
			// Select none
			m.selected = make(map[int]bool)
		}
	}
	return m, nil
}

func (m ReelListModel) View() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Select reels to transcribe:"))
	sb.WriteString("\n\n")

	for i, reel := range m.reels {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		checkbox := "[ ]"
		style := uncheckedStyle
		if m.selected[i] {
			checkbox = "[x]"
			style = checkedStyle
		}

		title := reel.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}

		views := formatViews(reel.ViewCount)
		line := fmt.Sprintf("%s %s %-42s %8s views", cursor, checkbox, title, views)
		sb.WriteString(style.Render(line))
		sb.WriteString("\n")
	}

	count := len(m.selected)
	sb.WriteString(fmt.Sprintf("\n%d selected | space=toggle, a=all, n=none, enter=confirm, q=cancel\n", count))

	return sb.String()
}

// SelectedReels returns the selected reels
func (m ReelListModel) SelectedReels() []*domain.Reel {
	var result []*domain.Reel
	for i, selected := range m.selected {
		if selected && i < len(m.reels) {
			result = append(result, m.reels[i])
		}
	}
	return result
}

// RunReelList displays the list and returns selected reels
func RunReelList(reels []*domain.Reel) ([]*domain.Reel, error) {
	if len(reels) == 0 {
		return nil, nil
	}

	model := NewReelListModel(reels)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	return finalModel.(ReelListModel).SelectedReels(), nil
}

func formatViews(count int64) string {
	if count >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	}
	if count >= 1000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%d", count)
}
```

**Step 4: Wire up interactive menu in root.go**

Update `runInteractiveMenu` in `internal/adapters/cli/root.go`:
```go
func runInteractiveMenu() error {
	options := []tui.MenuOption{
		{Label: "Transcribe a single reel", Value: "transcribe"},
		{Label: "Browse an account's reels", Value: "account"},
		{Label: "Manage cache", Value: "cache"},
		{Label: "Settings", Value: "settings"},
	}

	selected, err := tui.RunMenu(options)
	if err != nil {
		return err
	}

	switch selected {
	case "transcribe":
		fmt.Print("Enter reel URL or ID: ")
		var input string
		fmt.Scanln(&input)
		return runTranscribe(input)
	case "account":
		fmt.Print("Enter username: ")
		var username string
		fmt.Scanln(&username)
		return runAccountInteractive(username)
	case "cache":
		return runCacheInteractive()
	case "settings":
		fmt.Println("Settings not yet implemented")
	case "":
		fmt.Println("Cancelled")
	}

	return nil
}

func runAccountInteractive(username string) error {
	// TODO: Implement with BrowseService
	fmt.Printf("Browsing %s...\n", username)
	return nil
}

func runCacheInteractive() error {
	// TODO: Implement cache management
	fmt.Println("Cache management not yet implemented")
	return nil
}
```

Add import for tui package:
```go
import (
	// ... existing imports
	"github.com/devbush/ig2insights/internal/adapters/cli/tui"
)
```

**Step 5: Build and test**

Run:
```bash
go build -o ig2insights ./cmd/ig2insights && ./ig2insights
```

Expected: Interactive menu appears

**Step 6: Commit**

```bash
git add internal/adapters/cli/ go.mod go.sum
git commit -m "feat(cli): add interactive TUI with bubbletea"
```

---

## Phase 7: Integration & Polish

### Task 7.1: Implement Model Command

**Files:**
- Modify: `internal/adapters/cli/model.go`

**Step 1: Implement model list/download/remove**

Update `internal/adapters/cli/model.go`:
```go
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// NewModelCmd creates the model subcommand
func NewModelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Manage Whisper models",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available models",
		RunE:  runModelList,
	}

	downloadCmd := &cobra.Command{
		Use:   "download <model>",
		Short: "Download a model",
		Args:  cobra.ExactArgs(1),
		RunE:  runModelDownload,
	}

	removeCmd := &cobra.Command{
		Use:   "remove <model>",
		Short: "Remove a downloaded model",
		Args:  cobra.ExactArgs(1),
		RunE:  runModelRemove,
	}

	cmd.AddCommand(listCmd, downloadCmd, removeCmd)
	return cmd
}

func runModelList(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	models := app.Transcriber.AvailableModels()

	fmt.Println()
	fmt.Printf("  %-10s %-12s %s\n", "Model", "Size", "Status")
	fmt.Println("  " + strings.Repeat("─", 40))

	for _, m := range models {
		status := "not downloaded"
		if m.Downloaded {
			status = "✓ downloaded"
		}
		if m.Name == app.Config.Defaults.Model {
			status += " (default)"
		}

		size := formatSize(m.Size)
		fmt.Printf("  %-10s %-12s %s\n", m.Name, size, status)
	}
	fmt.Println()

	return nil
}

func runModelDownload(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	model := args[0]

	if app.Transcriber.IsModelDownloaded(model) {
		fmt.Printf("Model '%s' is already downloaded\n", model)
		return nil
	}

	fmt.Printf("Downloading model '%s'...\n", model)

	err = app.Transcriber.DownloadModel(context.Background(), model, func(downloaded, total int64) {
		if total > 0 {
			pct := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rProgress: %.1f%% (%s / %s)", pct, formatSize(downloaded), formatSize(total))
		}
	})

	if err != nil {
		return err
	}

	fmt.Println("\n✓ Model downloaded successfully")
	return nil
}

func runModelRemove(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	model := args[0]

	if !app.Transcriber.IsModelDownloaded(model) {
		fmt.Printf("Model '%s' is not downloaded\n", model)
		return nil
	}

	if err := app.Transcriber.DeleteModel(model); err != nil {
		return err
	}

	fmt.Printf("✓ Model '%s' removed\n", model)
	return nil
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.0f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.0f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
```

Add strings import at the top.

**Step 2: Commit**

```bash
git add internal/adapters/cli/model.go
git commit -m "feat(cli): implement model list/download/remove commands"
```

---

### Task 7.2: Implement Cache Command

**Files:**
- Modify: `internal/adapters/cli/cache.go`

**Step 1: Implement cache clear with stats**

Update `internal/adapters/cli/cache.go`:
```go
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var clearAllFlag bool

// NewCacheCmd creates the cache subcommand
func NewCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage cached transcripts",
		RunE:  runCacheStatus,
	}

	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear cache entries",
		RunE:  runCacheClear,
	}
	clearCmd.Flags().BoolVar(&clearAllFlag, "all", false, "Clear all cache entries")

	cmd.AddCommand(clearCmd)

	return cmd
}

func runCacheStatus(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	ctx := context.Background()
	stats, err := app.CacheSvc.Stats(ctx)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Cache Statistics:")
	fmt.Printf("  Items: %d\n", stats.ItemCount)
	fmt.Printf("  Size:  %s\n", formatSize(stats.TotalSize))
	fmt.Printf("  TTL:   %s\n", app.Config.Defaults.CacheTTL)
	fmt.Println()

	return nil
}

func runCacheClear(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	ctx := context.Background()

	if clearAllFlag {
		if err := app.CacheSvc.Clear(ctx); err != nil {
			return err
		}
		fmt.Println("✓ All cache entries cleared")
	} else {
		cleaned, err := app.CacheSvc.CleanExpired(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Removed %d expired entries\n", cleaned)
	}

	return nil
}
```

**Step 2: Commit**

```bash
git add internal/adapters/cli/cache.go
git commit -m "feat(cli): implement cache status and clear commands"
```

---

### Task 7.3: Implement Deps Command

**Files:**
- Modify: `internal/adapters/cli/deps.go`

**Step 1: Implement deps status/update**

Update `internal/adapters/cli/deps.go`:
```go
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// NewDepsCmd creates the deps subcommand
func NewDepsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deps",
		Short: "Manage dependencies (yt-dlp)",
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show dependency status",
		RunE:  runDepsStatus,
	}

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update yt-dlp to latest version",
		RunE:  runDepsUpdate,
	}

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install yt-dlp",
		RunE:  runDepsInstall,
	}

	cmd.AddCommand(statusCmd, updateCmd, installCmd)
	return cmd
}

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
		fmt.Printf("  yt-dlp:   ✓ installed (%s)\n", path)
	} else {
		fmt.Println("  yt-dlp:   ✗ not found")
	}

	// Whisper models
	models := app.Transcriber.AvailableModels()
	downloaded := 0
	for _, m := range models {
		if m.Downloaded {
			downloaded++
		}
	}
	fmt.Printf("  whisper:  %d/%d models downloaded\n", downloaded, len(models))
	fmt.Println()

	return nil
}

func runDepsUpdate(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	if !app.Downloader.IsAvailable() {
		return fmt.Errorf("yt-dlp is not installed. Run 'ig2insights deps install' first")
	}

	fmt.Println("Updating yt-dlp...")

	ctx := context.Background()
	if err := app.Downloader.Update(ctx); err != nil {
		return err
	}

	fmt.Println("✓ yt-dlp updated")
	return nil
}

func runDepsInstall(cmd *cobra.Command, args []string) error {
	app, err := GetApp()
	if err != nil {
		return err
	}

	if app.Downloader.IsAvailable() {
		fmt.Println("yt-dlp is already installed")
		return nil
	}

	fmt.Println("Installing yt-dlp...")

	ctx := context.Background()
	err = app.Downloader.Install(ctx, func(downloaded, total int64) {
		if total > 0 {
			pct := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rProgress: %.1f%%", pct)
		}
	})

	if err != nil {
		return err
	}

	fmt.Println("\n✓ yt-dlp installed")
	return nil
}
```

**Step 2: Commit**

```bash
git add internal/adapters/cli/deps.go
git commit -m "feat(cli): implement deps status/install/update commands"
```

---

### Task 7.4: Final Integration Test

**Step 1: Build the final binary**

Run:
```bash
go build -o ig2insights ./cmd/ig2insights
```

**Step 2: Test all commands**

Run:
```bash
./ig2insights --help
./ig2insights model list
./ig2insights deps status
./ig2insights cache
```

Expected: All commands work without errors

**Step 3: Final commit**

```bash
git add .
git commit -m "chore: final integration and cleanup"
```

---

## Summary

This plan implements ig2insights in 7 phases:

1. **Foundation** - Go module, domain entities, ports
2. **Configuration** - Config management, file paths
3. **Video Download** - yt-dlp adapter with auto-install
4. **Transcription** - Whisper adapter with model management
5. **Application Layer** - Use cases for transcribe, browse, cache
6. **CLI** - Cobra commands with bubbletea TUI
7. **Integration** - Wire everything together, polish

Each task follows TDD with bite-sized steps. Total: ~35 commits covering all functionality from the design document.
