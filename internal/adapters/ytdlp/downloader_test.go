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
