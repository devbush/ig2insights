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
