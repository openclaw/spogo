package atomicfile

import (
	"errors"
	"os"
	"path/filepath"
)

// Resolve aliases before locking and replacing the file so managed file
// symlinks keep their targets and concurrent aliases share the same lock.
func ResolvePath(path string) (string, error) {
	for range 255 {
		resolved, err := filepath.EvalSymlinks(path)
		if err == nil {
			return resolved, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		target, linkErr := os.Readlink(path)
		if linkErr != nil {
			dir, err := filepath.EvalSymlinks(filepath.Dir(path))
			if err != nil {
				return "", err
			}
			return filepath.Join(dir, filepath.Base(path)), nil
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		path = target
	}
	return "", errors.New("too many file symlinks")
}
