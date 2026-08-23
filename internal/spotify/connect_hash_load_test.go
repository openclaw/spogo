package spotify

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHashResolverLoad(t *testing.T) {
	hash := strings.Repeat("a", 64)
	mainJS := `var a={1:"web-player/main"};var b={1:"abcdef"};`
	chunkBody := `searchDesktop blah sha256Hash":"` + hash + `"`
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Host == "open.spotify.com":
			html := `<script src="https://open.spotifycdn.com/cdn/build/web-player/main.js"></script>`
			return textResponse(http.StatusOK, html), nil
		case strings.Contains(req.URL.Path, "/web-player/main.js"):
			return textResponse(http.StatusOK, mainJS), nil
		case strings.Contains(req.URL.Path, "web-player/main.abcdef.js"):
			return textResponse(http.StatusOK, chunkBody), nil
		default:
			return textResponse(http.StatusNotFound, "missing"), nil
		}
	})
	client := &http.Client{Transport: transport}
	resolver := newHashResolver(client, &connectSession{client: client})
	got, err := resolver.Hash(context.Background(), "searchDesktop")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if got != hash {
		t.Fatalf("unexpected hash: %s", got)
	}
}

func TestHashResolverLoadFromMainBody(t *testing.T) {
	hash := strings.Repeat("b", 64)
	mainJS := `"searchDesktop","query","` + hash + `"`
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Host == "open.spotify.com":
			html := `<script src="https://open.spotifycdn.com/cdn/build/web-player/main.js"></script>`
			return textResponse(http.StatusOK, html), nil
		case strings.Contains(req.URL.Path, "/web-player/main.js"):
			return textResponse(http.StatusOK, mainJS), nil
		default:
			return textResponse(http.StatusNotFound, "missing"), nil
		}
	})
	client := &http.Client{Transport: transport}
	resolver := newHashResolver(client, &connectSession{client: client})
	got, err := resolver.Hash(context.Background(), "searchDesktop")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if got != hash {
		t.Fatalf("unexpected hash: %s", got)
	}
}

func TestHashResolverPersistsHashesByClientVersion(t *testing.T) {
	cache := newConnectCacheStore(filepath.Join(t.TempDir(), "connect.json"))
	if err := cache.update(func(cached *connectCache) {
		cached.ClientVersion = "1.2.3"
	}); err != nil {
		t.Fatalf("seed cache: %v", err)
	}
	hashes := map[string]string{
		"getTrack":            strings.Repeat("a", 64),
		"getAlbum":            strings.Repeat("b", 64),
		"queryArtistOverview": strings.Repeat("c", 64),
		"searchDesktop":       strings.Repeat("d", 64),
	}
	var main strings.Builder
	for operation, hash := range hashes {
		fmt.Fprintf(&main, `"%s","query","%s";`, operation, hash)
	}
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case "open.spotify.com":
			return textResponse(http.StatusOK, `<script src="https://open.spotifycdn.com/cdn/build/web-player/main.js"></script>`), nil
		case "open.spotifycdn.com":
			return textResponse(http.StatusOK, main.String()), nil
		default:
			return nil, fmt.Errorf("unexpected host %q", req.URL.Host)
		}
	})
	client := &http.Client{Transport: transport}
	resolver := newHashResolver(client, &connectSession{client: client, cache: cache, clientVer: "1.2.3"})
	if got, err := resolver.Hash(context.Background(), "getTrack"); err != nil || got != hashes["getTrack"] {
		t.Fatalf("cold hash = %q, error = %v", got, err)
	}
	cached, err := cache.load()
	if err != nil {
		t.Fatalf("load cache: %v", err)
	}
	if cached.OperationHashesClientVersion != "1.2.3" || len(cached.OperationHashes) != len(hashes) {
		t.Fatalf("unexpected persisted hashes: version=%q hashes=%#v", cached.OperationHashesClientVersion, cached.OperationHashes)
	}
	warmClient := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("warm cache unexpectedly fetched %s", req.URL)
	})}
	warm := newHashResolver(warmClient, &connectSession{client: warmClient, cache: cache, clientVer: "1.2.3"})
	for operation, want := range hashes {
		if got, err := warm.Hash(context.Background(), operation); err != nil || got != want {
			t.Fatalf("warm %s hash = %q, error = %v", operation, got, err)
		}
	}
}

func TestHashResolverInvalidatesHashesWhenClientVersionChanges(t *testing.T) {
	cache := newConnectCacheStore(filepath.Join(t.TempDir(), "connect.json"))
	if err := cache.update(func(cached *connectCache) {
		cached.ClientVersion = "2.0.0"
		cached.OperationHashesClientVersion = "1.0.0"
		cached.OperationHashes = map[string]string{"getTrack": strings.Repeat("a", 64), "obsolete": "old"}
	}); err != nil {
		t.Fatalf("seed cache: %v", err)
	}
	want := strings.Repeat("b", 64)
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "open.spotify.com" {
			return textResponse(http.StatusOK, `<script src="https://open.spotifycdn.com/cdn/build/web-player/main.js"></script>`), nil
		}
		return textResponse(http.StatusOK, `"getTrack","query","`+want+`"`), nil
	})
	client := &http.Client{Transport: transport}
	resolver := newHashResolver(client, &connectSession{client: client, cache: cache, clientVer: "2.0.0"})
	if got, err := resolver.Hash(context.Background(), "getTrack"); err != nil || got != want {
		t.Fatalf("refreshed hash = %q, error = %v", got, err)
	}
	cached, err := cache.load()
	if err != nil {
		t.Fatalf("load cache: %v", err)
	}
	if cached.OperationHashesClientVersion != "2.0.0" || cached.OperationHashes["obsolete"] != "" {
		t.Fatalf("stale hashes survived build change: %#v", cached.OperationHashes)
	}
}

func TestHashResolverScansChunksWithBoundedConcurrency(t *testing.T) {
	const chunkCount = 16
	var names, hashes strings.Builder
	names.WriteByte('{')
	hashes.WriteByte('{')
	for index := 1; index <= chunkCount; index++ {
		if index > 1 {
			names.WriteByte(',')
			hashes.WriteByte(',')
		}
		fmt.Fprintf(&names, `%d:"chunk-%02d"`, index, index)
		fmt.Fprintf(&hashes, `%d:"%08x"`, index, index)
	}
	names.WriteByte('}')
	hashes.WriteByte('}')
	want := strings.Repeat("e", 64)
	var active, maximum atomic.Int32
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Host == "open.spotify.com":
			return textResponse(http.StatusOK, `<script src="https://open.spotifycdn.com/cdn/build/web-player/main.js"></script>`), nil
		case strings.HasSuffix(req.URL.Path, "/main.js"):
			return textResponse(http.StatusOK, "var names="+names.String()+";var hashes="+hashes.String()+";"), nil
		default:
			current := active.Add(1)
			for {
				previous := maximum.Load()
				if current <= previous || maximum.CompareAndSwap(previous, current) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			active.Add(-1)
			if strings.Contains(req.URL.Path, "chunk-16.") {
				return textResponse(http.StatusOK, `"getAlbum","query","`+want+`"`), nil
			}
			return textResponse(http.StatusOK, "no operations"), nil
		}
	})
	client := &http.Client{Transport: transport}
	resolver := newHashResolver(client, &connectSession{client: client})
	if got, err := resolver.Hash(context.Background(), "getAlbum"); err != nil || got != want {
		t.Fatalf("chunk hash = %q, error = %v", got, err)
	}
	if got := maximum.Load(); got <= 1 || got > hashChunkConcurrency {
		t.Fatalf("maximum chunk concurrency = %d, want between 2 and %d", got, hashChunkConcurrency)
	}
}
