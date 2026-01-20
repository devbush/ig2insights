package cli

import (
	"io"
	"os"
)

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	// If src and dst are the same, nothing to do
	if src == dst {
		return nil
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}

	_, err = io.Copy(destFile, sourceFile)
	if closeErr := destFile.Close(); err == nil {
		err = closeErr
	}
	return err
}
