package session

import (
	"fmt"
	"github.com/Sanidhya14321/goh/internal/platform"
	"io"
	"os"
	"path/filepath"
)

// WriteExportFile publishes complete private output at a new path.
func WriteExportFile(output string, write func(io.Writer) error) (err error) {
	output, err = filepath.Abs(output)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(output), ".harness-export-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err = write(file); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	// Publish complete bytes without replacing existing files or symlinks.
	if err = os.Link(file.Name(), output); err != nil {
		return fmt.Errorf("export destination must not exist: %w", err)
	}
	directory, err := os.Open(filepath.Dir(output))
	if err != nil {
		return fmt.Errorf("export published, but directory sync failed: %w", err)
	}
	defer directory.Close()
	if err := platform.SyncDir(directory); err != nil {
		return fmt.Errorf("export published, but directory sync failed: %w", err)
	}
	return nil
}
