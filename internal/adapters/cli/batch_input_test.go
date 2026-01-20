package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseInputFile(t *testing.T) {
	t.Run("parses file with comments, blank lines, URLs and IDs", func(t *testing.T) {
		// Create a temp file with mixed content
		content := `# This is a comment
https://www.instagram.com/reel/ABC123/
https://instagram.com/p/DEF456/

# Another comment
GHI789

XYZ999
`
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "input.txt")
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		ids, err := ParseInputFile(filePath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []string{"ABC123", "DEF456", "GHI789", "XYZ999"}
		if len(ids) != len(expected) {
			t.Fatalf("expected %d IDs, got %d: %v", len(expected), len(ids), ids)
		}

		for i, id := range ids {
			if id != expected[i] {
				t.Errorf("expected ID[%d] = %q, got %q", i, expected[i], id)
			}
		}
	})

	t.Run("returns error for nonexistent file", func(t *testing.T) {
		_, err := ParseInputFile("/nonexistent/path/file.txt")
		if err == nil {
			t.Error("expected error for nonexistent file, got nil")
		}
	})
}

func TestCollectInputs(t *testing.T) {
	t.Run("combines args and file with deduplication", func(t *testing.T) {
		// Create a temp file with some IDs
		content := `ABC123
https://www.instagram.com/reel/DEF456/
GHI789
`
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "input.txt")
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Args include ABC123 which is also in file (should be deduplicated)
		args := []string{"ABC123", "https://instagram.com/p/NEW001/"}

		ids, err := CollectInputs(args, filePath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Expected: ABC123, NEW001 from args, then DEF456, GHI789 from file (ABC123 deduplicated)
		expected := []string{"ABC123", "NEW001", "DEF456", "GHI789"}
		if len(ids) != len(expected) {
			t.Fatalf("expected %d IDs, got %d: %v", len(expected), len(ids), ids)
		}

		for i, id := range ids {
			if id != expected[i] {
				t.Errorf("expected ID[%d] = %q, got %q", i, expected[i], id)
			}
		}
	})

	t.Run("works with args only when filePath is empty", func(t *testing.T) {
		args := []string{"ABC123", "https://www.instagram.com/reel/DEF456/"}

		ids, err := CollectInputs(args, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []string{"ABC123", "DEF456"}
		if len(ids) != len(expected) {
			t.Fatalf("expected %d IDs, got %d: %v", len(expected), len(ids), ids)
		}

		for i, id := range ids {
			if id != expected[i] {
				t.Errorf("expected ID[%d] = %q, got %q", i, expected[i], id)
			}
		}
	})
}
