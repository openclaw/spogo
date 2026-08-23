package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/coder/websocket"
)

const (
	connectStateBase   = "https://gue1-spclient.spotify.com/connect-state/v1"
	trackPlaybackBase  = "https://gue1-spclient.spotify.com/track-playback/v1"
	connectionTTL      = 10 * time.Minute
	connectDeviceName  = "spogo"
	connectDeviceModel = "web_player"
)

var dealerURL = "wss://dealer.spotify.com/"

type connectState struct {
	raw            map[string]any
	playerState    map[string]any
	devices        map[string]any
	activeDeviceID string
	originDeviceID string
}

func (c *ConnectClient) connectState(ctx context.Context) (connectState, error) {
	auth, err := c.session.auth(ctx)
	if err != nil {
		return connectState{}, err
	}
	if err := c.ensureConnectDevice(ctx, auth); err != nil {
		return connectState{}, err
	}
	c.session.mu.Lock()
	deviceID := c.session.connectDeviceID
	connectionID := c.session.connectionID
	c.session.mu.Unlock()
	payload := map[string]any{
		"member_type": "CONNECT_STATE",
		"device": map[string]any{
			"device_info": map[string]any{
				"capabilities": map[string]any{
					"can_be_player":           false,
					"hidden":                  true,
					"needs_full_player_state": true,
				},
			},
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/devices/hobs_%s", connectStateBase, deviceID), encodeJSON(payload))
	if err != nil {
		return connectState{}, err
	}
	applyRequestHeaders(req, requestHeaders{
		AccessToken:   auth.AccessToken,
		ClientToken:   auth.ClientToken,
		ClientVersion: connectVersion(auth),
		ContentType:   "application/json",
		AppPlatform:   defaultSpotifyAppPlatform,
		ConnectionID:  connectionID,
	})
	resp, err := c.client.Do(req)
	if err != nil {
		return connectState{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return connectState{}, apiErrorFromResponse(resp)
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return connectState{}, err
	}
	state := connectState{raw: raw}
	if devices, ok := raw["devices"].(map[string]any); ok {
		state.devices = devices
	}
	if player, ok := raw["player_state"].(map[string]any); ok {
		state.playerState = player
	}
	if active, ok := raw["active_device_id"].(string); ok {
		state.activeDeviceID = active
	}
	if state.activeDeviceID == "" {
		state.activeDeviceID = detectActiveDeviceID(state.devices)
	}
	if origin := mapPlayOriginID(state.playerState); origin != "" {
		state.originDeviceID = origin
	}
	c.cacheCommandRoute(state)
	return state, nil
}

func (c *ConnectClient) ensureConnectDevice(ctx context.Context, auth connectAuth) error {
	c.session.mu.Lock()
	if c.session.connectDeviceID == "" {
		c.session.connectDeviceID = randomHex(32)
	}
	needs := c.session.connectionID == "" || time.Since(c.session.registeredAt) > connectionTTL
	c.session.mu.Unlock()
	if !needs {
		return nil
	}
	connectionID, err := getConnectionID(ctx, auth.AccessToken, c.client.Timeout)
	if err != nil {
		return err
	}
	if err := c.registerDevice(ctx, auth, connectionID); err != nil {
		return err
	}
	c.session.mu.Lock()
	c.session.connectionID = connectionID
	c.session.registeredAt = time.Now()
	c.session.saveCacheLocked()
	c.session.mu.Unlock()
	return nil
}

func (c *ConnectClient) registerDevice(ctx context.Context, auth connectAuth, connectionID string) error {
	c.session.mu.Lock()
	deviceID := c.session.connectDeviceID
	c.session.mu.Unlock()
	payload := map[string]any{
		"device": map[string]any{
			"device_id":           deviceID,
			"device_type":         "computer",
			"brand":               "spotify",
			"model":               connectDeviceModel,
			"name":                connectDeviceName,
			"is_group":            false,
			"metadata":            map[string]any{},
			"platform_identifier": fmt.Sprintf("web_player %s;spogo", runtime.GOOS),
			"capabilities": map[string]any{
				"change_volume":            true,
				"supports_file_media_type": true,
				"enable_play_token":        true,
				"play_token_lost_behavior": "pause",
				"disable_connect":          false,
				"audio_podcasts":           true,
				"video_playback":           true,
				"manifest_formats": []string{
					"file_ids_mp3",
					"file_urls_mp3",
					"file_ids_mp4",
					"manifest_ids_video",
				},
			},
		},
		"outro_endcontent_snooping": false,
		"connection_id":             connectionID,
		"client_version":            connectVersion(auth),
		"volume":                    65535,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, trackPlaybackBase+"/devices", encodeJSON(payload))
	if err != nil {
		return err
	}
	applyRequestHeaders(req, requestHeaders{
		AccessToken:   auth.AccessToken,
		ClientToken:   auth.ClientToken,
		ClientVersion: connectVersion(auth),
		ContentType:   "application/json",
		AppPlatform:   defaultSpotifyAppPlatform,
	})
	resp, err := c.client.Do(req)
	if err != nil {
		c.invalidateCommandRoute()
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.invalidateCommandRoute()
		return apiErrorFromResponse(resp)
	}
	return nil
}

func (c *ConnectClient) sendPlayerCommand(ctx context.Context, state connectState, endpoint string, payload map[string]any) error {
	if payload == nil {
		payload = map[string]any{
			"command": map[string]any{
				"endpoint": endpoint,
				"logging_params": map[string]any{
					"command_id": randomHex(32),
				},
			},
		}
	}
	fromID := state.originDeviceID
	if fromID == "" {
		c.session.mu.Lock()
		fromID = c.session.connectDeviceID
		c.session.mu.Unlock()
	}
	if fromID == "" {
		fromID = state.activeDeviceID
	}
	if fromID == "" || state.activeDeviceID == "" {
		return errors.New("missing device id")
	}
	url := fmt.Sprintf("%s/player/command/from/%s/to/%s", connectStateBase, fromID, state.activeDeviceID)
	return c.sendConnectCommand(ctx, url, payload)
}

func (c *ConnectClient) sendConnectCommand(ctx context.Context, url string, payload map[string]any) error {
	return c.sendConnectRequest(ctx, http.MethodPost, url, payload)
}

func (c *ConnectClient) sendConnectRequest(ctx context.Context, method, url string, payload map[string]any) error {
	auth, err := c.session.auth(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, url, encodeJSON(payload))
	if err != nil {
		return err
	}
	applyRequestHeaders(req, requestHeaders{
		AccessToken:   auth.AccessToken,
		ClientToken:   auth.ClientToken,
		ClientVersion: connectVersion(auth),
		ContentType:   "application/json",
		AppPlatform:   defaultSpotifyAppPlatform,
	})
	resp, err := c.client.Do(req)
	if err != nil {
		c.invalidateCommandRoute()
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.invalidateCommandRoute()
		return apiErrorFromResponse(resp)
	}
	return nil
}

func getConnectionID(ctx context.Context, accessToken string, timeout time.Duration) (string, error) {
	ctx, cancel := dealerConnectionContext(ctx, timeout)
	defer cancel()
	url := dealerURL
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
		if strings.HasSuffix(url, "?") || strings.HasSuffix(url, "&") {
			sep = ""
		}
	}
	url += sep + "access_token=" + accessToken
	conn, resp, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"User-Agent": []string{defaultUserAgent()},
		},
	})
	if err != nil {
		return "", redactedDealerError{err: err, accessToken: accessToken}
	}
	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	defer func() {
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()
	_, data, err := conn.Read(ctx)
	if err != nil {
		return "", err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", err
	}
	headers, ok := payload["headers"].(map[string]any)
	if !ok {
		return "", errors.New("missing headers")
	}
	for key, value := range headers {
		if !strings.EqualFold(key, "Spotify-Connection-Id") {
			continue
		}
		if id, ok := value.(string); ok && id != "" {
			return id, nil
		}
	}
	return "", errors.New("missing connection id")
}

func dealerConnectionContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		timeout = defaultHTTPClientTimeout
	}
	if timeout < 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

type redactedDealerError struct {
	err         error
	accessToken string
}

func (e redactedDealerError) Error() string {
	if e.accessToken == "" {
		return e.err.Error()
	}
	return strings.ReplaceAll(e.err.Error(), e.accessToken, "[REDACTED]")
}

func (e redactedDealerError) Unwrap() error {
	return e.err
}
