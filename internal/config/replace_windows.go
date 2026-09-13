//go:build windows

package config

import (
	"errors"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

func replaceConfigFile(source, destination string) error {
	// Windows reports either sharing violation or access denied when a reader
	// does not share deletion. Bound retries so permanent permission errors return.
	for attempt := 0; ; attempt++ {
		err := os.Rename(source, destination)
		busy := errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_ACCESS_DENIED)
		if !busy || attempt >= 100 {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
}
