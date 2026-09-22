// Package atomicfile replaces private files without exposing partial contents.
package atomicfile

import (
	"os"
	"path/filepath"
)

// Write commits data from a synced, owner-only temporary file in the same directory.
// The caller owns directory creation and any transaction locking.
func Write(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".spogo-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}()
	if err := file.Chmod(0o600); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return replace(file.Name(), path)
}
