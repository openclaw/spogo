package spotify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gofrs/flock"
)

const oauthLockRetryDelay = 25 * time.Millisecond

func WithOAuthLifecycleLock(ctx context.Context, path string, fn func() error) error {
	if path == "" {
		return errors.New("oauth token cache path is required")
	}
	lifecycleLock, err := acquireOAuthLifecycleLock(ctx, path)
	if err != nil {
		return err
	}
	defer releaseOAuthCacheLock(lifecycleLock)
	return fn()
}

func acquireOAuthCacheLock(ctx context.Context, path string) (*flock.Flock, error) {
	if path == "" {
		return nil, errors.New("oauth token cache path is required")
	}
	return acquireOAuthFileLock(ctx, path+".lock", filepath.Dir(path), "oauth token cache")
}

func acquireOAuthLifecycleLock(ctx context.Context, path string) (*flock.Flock, error) {
	return acquireOAuthFileLock(ctx, path+".lifecycle.lock", filepath.Dir(path), "oauth lifecycle")
}

func acquireOAuthFileLock(ctx context.Context, lockPath, dir, label string) (*flock.Flock, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	cacheLock := flock.New(lockPath, flock.SetPermissions(0o600))
	locked, err := cacheLock.TryLockContext(ctx, oauthLockRetryDelay)
	if err != nil {
		_ = cacheLock.Close()
		return nil, fmt.Errorf("lock %s: %w", label, err)
	}
	if !locked {
		_ = cacheLock.Close()
		return nil, fmt.Errorf("lock %s: %w", label, ctx.Err())
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(cacheLock.Path(), 0o600); err != nil {
			releaseOAuthCacheLock(cacheLock)
			return nil, err
		}
	}
	return cacheLock, nil
}

func releaseOAuthCacheLock(cacheLock *flock.Flock) {
	if cacheLock == nil {
		return
	}
	_ = cacheLock.Unlock()
	_ = cacheLock.Close()
}
