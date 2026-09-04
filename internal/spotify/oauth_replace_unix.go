//go:build !windows

package spotify

import "os"

func replaceOAuthTokenFile(source, destination string) error {
	return os.Rename(source, destination)
}
