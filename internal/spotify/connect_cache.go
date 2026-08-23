package spotify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	connectCacheVersion = 1
	commandRouteTTL     = 10 * time.Minute
)

type connectCache struct {
	Version int `json:"version"`

	AccessToken            string `json:"access_token,omitempty"`
	AccessTokenExpiresUnix int64  `json:"access_token_expires_unix,omitempty"`
	Anonymous              bool   `json:"anonymous,omitempty"`
	ClientID               string `json:"client_id,omitempty"`

	ClientToken            string `json:"client_token,omitempty"`
	ClientTokenExpiresUnix int64  `json:"client_token_expires_unix,omitempty"`
	ClientVersion          string `json:"client_version,omitempty"`
	ConnectVersion         string `json:"connect_version,omitempty"`
	DeviceID               string `json:"device_id,omitempty"`
	ConnectDeviceID        string `json:"connect_device_id,omitempty"`
	// GraphQL operation hashes belong to the web-player client build.
	OperationHashesClientVersion string            `json:"operation_hashes_client_version,omitempty"`
	OperationHashes              map[string]string `json:"operation_hashes,omitempty"`

	ActiveDeviceID string `json:"active_device_id,omitempty"`
	OriginDeviceID string `json:"origin_device_id,omitempty"`
	RouteUnix      int64  `json:"route_unix,omitempty"`
}

type connectCacheStore struct {
	path string
	mu   sync.Mutex
}

func newConnectCacheStore(path string) *connectCacheStore {
	if path == "" {
		return nil
	}
	return &connectCacheStore{path: path}
}

func (s *connectCacheStore) load() (connectCache, error) {
	if s == nil || s.path == "" {
		return connectCache{}, os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *connectCacheStore) update(fn func(*connectCache)) error {
	if s == nil || s.path == "" || fn == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cache := connectCache{Version: connectCacheVersion}
	if loaded, err := s.loadLocked(); err == nil {
		cache = loaded
	}
	fn(&cache)
	cache.Version = connectCacheVersion
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func (s *connectCacheStore) loadLocked() (connectCache, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return connectCache{}, err
	}
	var cache connectCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return connectCache{}, err
	}
	if cache.Version != connectCacheVersion {
		return connectCache{}, os.ErrNotExist
	}
	return cache, nil
}

func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func timeFromUnix(unix int64) time.Time {
	if unix <= 0 {
		return time.Time{}
	}
	return time.Unix(unix, 0)
}

func (c *ConnectClient) cacheCommandRoute(state connectState) {
	if c == nil || state.activeDeviceID == "" {
		return
	}
	now := time.Now()
	c.routeMu.Lock()
	c.cachedActiveDeviceID = state.activeDeviceID
	c.cachedOriginDeviceID = state.originDeviceID
	c.cachedRouteAt = now
	c.routeMu.Unlock()
	if c.cache != nil {
		active := state.activeDeviceID
		origin := state.originDeviceID
		c.session.mu.Lock()
		connectDeviceID := c.session.connectDeviceID
		c.session.mu.Unlock()
		_ = c.cache.update(func(cache *connectCache) {
			cache.ConnectDeviceID = connectDeviceID
			cache.ActiveDeviceID = active
			cache.OriginDeviceID = origin
			cache.RouteUnix = now.Unix()
		})
	}
}

func (c *ConnectClient) commandRoute() (string, string, bool) {
	if c == nil {
		return "", "", false
	}
	if from, to, ok := c.memoryCommandRoute(); ok {
		return from, to, true
	}
	if c.cache == nil {
		return "", "", false
	}
	cached, err := c.cache.load()
	if err != nil {
		return "", "", false
	}
	routeAt := timeFromUnix(cached.RouteUnix)
	if cached.ActiveDeviceID == "" || routeAt.IsZero() || time.Since(routeAt) > commandRouteTTL {
		return "", "", false
	}
	c.routeMu.Lock()
	c.cachedActiveDeviceID = cached.ActiveDeviceID
	c.cachedOriginDeviceID = cached.OriginDeviceID
	c.cachedRouteAt = routeAt
	c.routeMu.Unlock()
	if cached.ConnectDeviceID != "" {
		c.session.mu.Lock()
		if c.session.connectDeviceID == "" {
			c.session.connectDeviceID = cached.ConnectDeviceID
		}
		c.session.mu.Unlock()
	}
	return c.memoryCommandRoute()
}

func (c *ConnectClient) memoryCommandRoute() (string, string, bool) {
	c.routeMu.RLock()
	active := c.cachedActiveDeviceID
	origin := c.cachedOriginDeviceID
	at := c.cachedRouteAt
	c.routeMu.RUnlock()
	if active == "" || at.IsZero() || time.Since(at) > commandRouteTTL {
		return "", "", false
	}
	from := origin
	if from == "" {
		c.session.mu.Lock()
		from = c.session.connectDeviceID
		c.session.mu.Unlock()
	}
	if from == "" {
		from = active
	}
	if from == "" {
		return "", "", false
	}
	return from, active, true
}

func (c *ConnectClient) invalidateCommandRoute() {
	if c == nil {
		return
	}
	c.routeMu.Lock()
	c.cachedActiveDeviceID = ""
	c.cachedOriginDeviceID = ""
	c.cachedRouteAt = time.Time{}
	c.routeMu.Unlock()
	if c.cache != nil {
		_ = c.cache.update(func(cache *connectCache) {
			cache.ActiveDeviceID = ""
			cache.OriginDeviceID = ""
			cache.RouteUnix = 0
		})
	}
}
