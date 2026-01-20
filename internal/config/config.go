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
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
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
