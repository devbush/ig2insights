# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ig2insights is a Go CLI tool that transcribes Instagram Reels to text using local Whisper and yt-dlp for video downloading.

## Build & Test Commands

```bash
# Build
go build -o ig2insights ./cmd/ig2insights

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./internal/domain/...

# Cross-compile
GOOS=darwin GOARCH=amd64 go build -o ig2insights-darwin ./cmd/ig2insights
GOOS=linux GOARCH=amd64 go build -o ig2insights-linux ./cmd/ig2insights
```

## Architecture

This project uses **Domain-Driven Design with Hexagonal Architecture (Ports & Adapters)**.

### Layer Structure

```
internal/
├── domain/        # Pure business entities (Reel, Account, Transcript) - NO external deps
├── ports/         # Interface definitions (Transcriber, VideoDownloader, CacheStore)
├── application/   # Use cases that orchestrate domain + ports (TranscribeService, BrowseService)
├── adapters/      # Concrete implementations
│   ├── cli/       # Cobra commands + Bubbletea TUI components
│   ├── ytdlp/     # Implements VideoDownloader port
│   ├── whisper/   # Implements Transcriber port
│   └── cache/     # Implements CacheStore port
└── config/        # Configuration loading and path management
```

### Key Principle

**Dependency direction flows inward**: CLI → Application → Ports ← Adapters, with Domain at the center having no external dependencies.

### Adding Features

- **New domain logic**: Add to `internal/domain/` (keep it pure, no external deps)
- **New external integration**: Create adapter in `internal/adapters/`, implement a port interface
- **New use case**: Add service in `internal/application/`
- **New CLI command**: Add to `internal/adapters/cli/`

## Key Files

- `cmd/ig2insights/main.go` - Entry point, delegates to CLI adapter
- `internal/adapters/cli/root.go` - CLI orchestration and command setup
- `internal/adapters/cli/app.go` - Dependency injection
- `internal/application/transcribe.go` - Main transcription workflow
- `internal/config/config.go` - Config loading, paths, directory setup

## Configuration

User config stored at `~/.ig2insights/config.yaml`. The config module handles:
- Auto-downloading dependencies (yt-dlp, ffmpeg on Windows)
- Whisper model management in `~/.ig2insights/models/`
- Transcript caching in `~/.ig2insights/cache/`

## Dependencies

- **CLI**: `spf13/cobra`
- **TUI**: `charmbracelet/bubbletea`, `charmbracelet/bubbles`, `charmbracelet/lipgloss`
- **Config**: `gopkg.in/yaml.v3`
- **Archive extraction**: `bodgit/sevenzip` and related packages

## Design Documentation

Detailed design and implementation plans are in `docs/plans/`. Reference these when making significant changes to understand the intended architecture.
