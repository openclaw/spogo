package spotify

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"
	"sync"
)

type hashResolver struct {
	client  *http.Client
	session *connectSession

	mu            sync.Mutex
	loadMu        sync.Mutex
	hashes        map[string]string
	clientVersion string
}

const hashChunkConcurrency = 8

var commonGraphQLOperations = []string{
	"getTrack", "getAlbum", "queryArtistOverview", "searchDesktop", "queryPodcastEpisodes", "getEpisodeOrChapter",
}

func newHashResolver(client *http.Client, session *connectSession) *hashResolver {
	return &hashResolver{
		client:  client,
		session: session,
		hashes:  map[string]string{},
	}
}

func (h *hashResolver) Hash(ctx context.Context, operation string) (string, error) {
	if operation == "" {
		return "", errors.New("operation required")
	}
	h.loadCachedHashes()
	h.mu.Lock()
	if hash, ok := h.hashes[operation]; ok && hash != "" {
		h.mu.Unlock()
		return hash, nil
	}
	h.mu.Unlock()
	if err := h.load(ctx, []string{operation}); err != nil {
		return "", err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	hash := h.hashes[operation]
	if hash == "" {
		return "", fmt.Errorf("hash for %s not found", operation)
	}
	return hash, nil
}

func (h *hashResolver) load(ctx context.Context, ops []string) error {
	h.loadMu.Lock()
	defer h.loadMu.Unlock()
	h.loadCachedHashes()
	h.mu.Lock()
	need := make([]string, 0, len(ops))
	for _, op := range ops {
		if h.hashes[op] == "" {
			need = append(need, op)
		}
	}
	if len(need) == 0 {
		h.mu.Unlock()
		return nil
	}
	wanted := append([]string(nil), need...)
	for _, op := range commonGraphQLOperations {
		if h.hashes[op] == "" && !slices.Contains(wanted, op) {
			wanted = append(wanted, op)
		}
	}
	h.mu.Unlock()
	html, err := h.fetchWebPlayerHTML(ctx)
	if err != nil {
		return err
	}
	mainJS, err := pickWebPlayerBundle(html)
	if err != nil {
		return err
	}
	bundleBase := bundleBaseURL(mainJS)
	mainBody, err := h.fetchText(ctx, mainJS)
	if err != nil {
		return err
	}
	if found := findOperationHashes(mainBody, wanted); len(found) > 0 {
		h.storeHashes(found)
		need = filterMissing(need, found)
		wanted = filterMissing(wanted, found)
		if len(wanted) == 0 {
			return nil
		}
	}
	nameMap, hashMap, err := parseWebpackMaps(mainBody)
	if err != nil {
		if len(need) == 0 {
			return nil
		}
		return err
	}
	chunks := prioritizeOperationChunks(combineChunkNames(nameMap, hashMap), wanted)
	if len(chunks) == 0 {
		if len(need) == 0 {
			return nil
		}
		return errors.New("no chunks found")
	}
	scanCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	for found := range h.scanChunks(scanCtx, bundleBase, chunks, wanted) {
		h.storeHashes(found)
		need = filterMissing(need, found)
		wanted = filterMissing(wanted, found)
		if len(wanted) == 0 {
			return nil
		}
	}
	if len(need) == 0 {
		return nil
	}
	return fmt.Errorf("missing hashes for %s", strings.Join(need, ", "))
}

func (h *hashResolver) loadCachedHashes() {
	if h.session == nil || h.session.cache == nil {
		return
	}
	h.session.mu.Lock()
	version := h.session.clientVer
	h.session.mu.Unlock()
	if version == "" {
		return
	}
	h.mu.Lock()
	if h.clientVersion == version {
		h.mu.Unlock()
		return
	}
	if h.clientVersion != "" {
		h.hashes = map[string]string{}
	}
	h.clientVersion = version
	h.mu.Unlock()
	cached, err := h.session.cache.load()
	if err != nil || cached.OperationHashesClientVersion != version {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for operation, hash := range cached.OperationHashes {
		if h.hashes[operation] == "" {
			h.hashes[operation] = hash
		}
	}
}

func (h *hashResolver) storeHashes(found map[string]string) {
	h.mu.Lock()
	for operation, hash := range found {
		if h.hashes[operation] == "" {
			h.hashes[operation] = hash
		}
	}
	version := h.clientVersion
	h.mu.Unlock()
	if version == "" || h.session == nil || h.session.cache == nil {
		return
	}
	_ = h.session.cache.update(func(cached *connectCache) {
		if cached.OperationHashesClientVersion != version {
			cached.OperationHashesClientVersion = version
			cached.OperationHashes = map[string]string{}
		}
		if cached.OperationHashes == nil {
			cached.OperationHashes = map[string]string{}
		}
		for operation, hash := range found {
			cached.OperationHashes[operation] = hash
		}
	})
}

func (h *hashResolver) scanChunks(ctx context.Context, base string, chunks, operations []string) <-chan map[string]string {
	results := make(chan map[string]string, hashChunkConcurrency)
	jobs := make(chan string)
	ctx, cancel := context.WithCancel(ctx)
	var workers sync.WaitGroup
	for range min(hashChunkConcurrency, len(chunks)) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case chunk, ok := <-jobs:
					if !ok {
						return
					}
					body, err := h.fetchText(ctx, base+chunk)
					if err != nil {
						continue
					}
					if found := findOperationHashes(body, operations); len(found) > 0 {
						select {
						case results <- found:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, chunk := range chunks {
			select {
			case jobs <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		workers.Wait()
		cancel()
		close(results)
	}()
	return results
}

func prioritizeOperationChunks(chunks, operations []string) []string {
	score := func(chunk string) int {
		chunk = strings.ToLower(chunk)
		best := 0
		for _, operation := range operations {
			var term string
			switch {
			case strings.Contains(strings.ToLower(operation), "search"):
				term = "search"
			case strings.Contains(strings.ToLower(operation), "album"):
				term = "album"
			case strings.Contains(strings.ToLower(operation), "artist"):
				term = "artist"
			case strings.Contains(strings.ToLower(operation), "episode"):
				term = "episode"
			case strings.Contains(strings.ToLower(operation), "podcast"):
				term = "podcast"
			}
			if term == "" || !strings.Contains(chunk, term) {
				continue
			}
			if strings.Contains(chunk, "routes-"+term+".") {
				best = max(best, 2)
			} else {
				best = max(best, 1)
			}
		}
		return best
	}
	sort.SliceStable(chunks, func(i, j int) bool { return score(chunks[i]) > score(chunks[j]) })
	return chunks
}

func (h *hashResolver) fetchWebPlayerHTML(ctx context.Context) (string, error) {
	return h.fetchText(ctx, "https://open.spotify.com/")
}

func (h *hashResolver) fetchText(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	applyRequestHeaders(req, requestHeaders{})
	resp, err := h.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", apiErrorFromResponse(resp)
	}
	body, err := readAll(resp)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func filterMissing(need []string, found map[string]string) []string {
	remaining := make([]string, 0, len(need))
	for _, op := range need {
		if found[op] == "" {
			remaining = append(remaining, op)
		}
	}
	return remaining
}
