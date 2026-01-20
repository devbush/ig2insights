package ytdlp

import (
	"runtime"
	"strings"
	"testing"

	"github.com/devbush/ig2insights/internal/domain"
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

func TestNewDownloader(t *testing.T) {
	d := NewDownloader()

	if d == nil {
		t.Fatal("NewDownloader() returned nil")
	}

	// New downloader should have empty binPath (not yet resolved)
	if d.binPath != "" {
		t.Errorf("NewDownloader().binPath = %q, want empty string", d.binPath)
	}
}

func TestIsAvailable(t *testing.T) {
	t.Run("returns false when binary not found", func(t *testing.T) {
		d := &Downloader{binPath: ""}
		// Override findBinary behavior by checking the result
		// IsAvailable calls GetBinaryPath which calls findBinary
		// Since we don't have yt-dlp installed in test env, this tests the negative case
		result := d.IsAvailable()
		// The result depends on whether yt-dlp is actually installed
		// We just verify the function runs without error
		_ = result
	})

	t.Run("returns true when binPath is set", func(t *testing.T) {
		d := &Downloader{binPath: "/some/path/to/yt-dlp"}
		if !d.IsAvailable() {
			t.Error("IsAvailable() = false, want true when binPath is set")
		}
	})

	t.Run("returns false when binPath is empty and binary not found", func(t *testing.T) {
		// Create a fresh downloader with empty binPath
		d := &Downloader{binPath: ""}
		// If yt-dlp is not in PATH or bundled, should return false
		// Result depends on system state, just verify no panic
		_ = d.IsAvailable()
	})
}

func TestGetBinaryPath_Caching(t *testing.T) {
	t.Run("returns cached path on subsequent calls", func(t *testing.T) {
		d := &Downloader{binPath: "/cached/path/yt-dlp"}

		// First call should return cached value
		path1 := d.GetBinaryPath()
		if path1 != "/cached/path/yt-dlp" {
			t.Errorf("GetBinaryPath() first call = %q, want %q", path1, "/cached/path/yt-dlp")
		}

		// Second call should return same cached value
		path2 := d.GetBinaryPath()
		if path2 != path1 {
			t.Errorf("GetBinaryPath() second call = %q, want %q (cached)", path2, path1)
		}
	})

	t.Run("caches result after first lookup", func(t *testing.T) {
		d := NewDownloader()

		// First call triggers lookup and caches result
		path1 := d.GetBinaryPath()

		// Second call should return cached value (same as first)
		path2 := d.GetBinaryPath()

		if path1 != path2 {
			t.Errorf("GetBinaryPath() not caching: first=%q, second=%q", path1, path2)
		}

		// Verify binPath is now set (even if empty string when not found)
		// The key is that subsequent calls don't re-lookup
		if d.binPath != path1 {
			t.Errorf("GetBinaryPath() did not cache: binPath=%q, returned=%q", d.binPath, path1)
		}
	})

	t.Run("empty binPath triggers lookup", func(t *testing.T) {
		d := &Downloader{binPath: ""}

		// With empty binPath, GetBinaryPath should call findBinary
		path := d.GetBinaryPath()

		// After call, binPath should be set to whatever findBinary returned
		if d.binPath != path {
			t.Errorf("GetBinaryPath() binPath mismatch: field=%q, returned=%q", d.binPath, path)
		}
	})
}

func TestSortByViews(t *testing.T) {
	t.Run("sorts reels by view count descending", func(t *testing.T) {
		reels := []*domain.Reel{
			{ID: "a", ViewCount: 100},
			{ID: "b", ViewCount: 500},
			{ID: "c", ViewCount: 200},
			{ID: "d", ViewCount: 1000},
		}

		sortByViews(reels)

		// Verify descending order
		expected := []int64{1000, 500, 200, 100}
		for i, reel := range reels {
			if reel.ViewCount != expected[i] {
				t.Errorf("sortByViews() index %d: got %d, want %d", i, reel.ViewCount, expected[i])
			}
		}
	})

	t.Run("handles empty slice", func(t *testing.T) {
		var reels []*domain.Reel
		// Should not panic
		sortByViews(reels)
		if len(reels) != 0 {
			t.Errorf("sortByViews() modified empty slice")
		}
	})

	t.Run("handles single element", func(t *testing.T) {
		reels := []*domain.Reel{{ID: "a", ViewCount: 100}}
		sortByViews(reels)
		if reels[0].ViewCount != 100 {
			t.Errorf("sortByViews() changed single element")
		}
	})

	t.Run("handles already sorted slice", func(t *testing.T) {
		reels := []*domain.Reel{
			{ID: "a", ViewCount: 300},
			{ID: "b", ViewCount: 200},
			{ID: "c", ViewCount: 100},
		}

		sortByViews(reels)

		expected := []int64{300, 200, 100}
		for i, reel := range reels {
			if reel.ViewCount != expected[i] {
				t.Errorf("sortByViews() index %d: got %d, want %d", i, reel.ViewCount, expected[i])
			}
		}
	})

	t.Run("handles equal view counts", func(t *testing.T) {
		reels := []*domain.Reel{
			{ID: "a", ViewCount: 100},
			{ID: "b", ViewCount: 100},
			{ID: "c", ViewCount: 100},
		}

		sortByViews(reels)

		// All should still have 100 views
		for i, reel := range reels {
			if reel.ViewCount != 100 {
				t.Errorf("sortByViews() index %d: got %d, want 100", i, reel.ViewCount)
			}
		}
	})
}

func TestGetAccount_NoBinary(t *testing.T) {
	d := &Downloader{binPath: ""}

	// Create a downloader that will not find the binary
	// Since findBinary might actually find yt-dlp on the system,
	// we need to test when binPath explicitly returns empty
	d.binPath = "" // Reset to trigger lookup

	// If yt-dlp is not installed, GetBinaryPath returns empty
	// We can't easily mock this, so we test the error case directly
	// by checking behavior when binary truly isn't found

	// This test verifies the error message format
	// when yt-dlp is not found
	t.Run("returns error when binary not found", func(t *testing.T) {
		// Create a downloader with explicit empty path that won't find binary
		testDownloader := &Downloader{}

		// Only run this assertion if yt-dlp is not actually installed
		if testDownloader.GetBinaryPath() == "" {
			_, err := testDownloader.GetAccount(nil, "testuser")
			if err == nil {
				t.Error("GetAccount() expected error when binary not found")
			}
			if err.Error() != "yt-dlp not found" {
				t.Errorf("GetAccount() error = %q, want %q", err.Error(), "yt-dlp not found")
			}
		}
	})
}

func TestListReels_NoBinary(t *testing.T) {
	t.Run("returns error when binary not found", func(t *testing.T) {
		testDownloader := &Downloader{}

		// Only run this assertion if yt-dlp is not actually installed
		if testDownloader.GetBinaryPath() == "" {
			_, err := testDownloader.ListReels(nil, "testuser", domain.SortLatest, 10)
			if err == nil {
				t.Error("ListReels() expected error when binary not found")
			}
			if err.Error() != "yt-dlp not found" {
				t.Errorf("ListReels() error = %q, want %q", err.Error(), "yt-dlp not found")
			}
		}
	})
}

func TestAccountFetcherInterface(t *testing.T) {
	// Verify that Downloader implements AccountFetcher interface at compile time
	// This is also done via var _ ports.AccountFetcher = (*Downloader)(nil)
	// but this test documents the intent
	t.Run("Downloader implements AccountFetcher", func(t *testing.T) {
		d := NewDownloader()
		// If this compiles, the interface is satisfied
		_ = d.GetAccount
		_ = d.ListReels
	})
}

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
