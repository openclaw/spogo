package spotify

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func (c *ConnectClient) playback(ctx context.Context) (PlaybackStatus, error) {
	return withConnectState(ctx, c, func(state connectState) (PlaybackStatus, error) {
		status := mapPlaybackStatus(state)
		if itemNeedsTrackMetadata(status.Item) {
			if full, err := c.trackInfo(ctx, status.Item.ID); err == nil {
				mergeItemMetadata(status.Item, full)
			}
		}
		return status, nil
	})
}

func (c *ConnectClient) devices(ctx context.Context) ([]Device, error) {
	return withConnectState(ctx, c, func(state connectState) ([]Device, error) {
		return mapDevices(state), nil
	})
}

func (c *ConnectClient) transfer(ctx context.Context, deviceID string) error {
	return withConnectStateErr(ctx, c, func(state connectState) error {
		fromID := connectTransferSourceID(state)
		if fromID == "" {
			return c.transferViaWebAPI(ctx, deviceID)
		}
		return c.sendConnectCommand(ctx, fmt.Sprintf("%s/connect/transfer/from/%s/to/%s", connectStateBase, fromID, deviceID), map[string]any{
			"transfer_options": map[string]any{
				"restore_paused": "resume",
			},
			"command_id": randomHex(32),
		})
	})
}

func (c *ConnectClient) transferViaWebAPI(ctx context.Context, deviceID string) error {
	return withWebFallback(c, func(web *Client) error {
		return web.Transfer(ctx, deviceID)
	})
}

func (c *ConnectClient) play(ctx context.Context, uri string) error {
	return withConnectStateErr(ctx, c, func(state connectState) error {
		if state.activeDeviceID == "" {
			if targetID := resolveConnectTargetDeviceID(state, c.device); targetID != "" {
				state.activeDeviceID = targetID
			} else {
				return c.playViaWebAPI(ctx, uri)
			}
		}
		if uri == "" {
			return c.sendPlayerCommand(ctx, state, "resume", nil)
		}
		return c.sendPlayerCommand(ctx, state, "play", playCommandPayload(uri))
	})
}

func (c *ConnectClient) playViaWebAPI(ctx context.Context, uri string) error {
	return withWebFallback(c, func(web *Client) error {
		return web.Play(ctx, uri)
	})
}

// playTracks starts an ordered list of track URIs. The Connect play command
// models a single context plus an optional skip_to target, so it cannot cleanly
// represent an ad-hoc ordered track list. Zero/one URIs keep the native Connect
// path (resume / single play); two or more delegate to the Web API's
// /me/player/play endpoint, which natively accepts an ordered uris array. The
// web fallback carries the same device targeting as the single-track fallback.
func (c *ConnectClient) playTracks(ctx context.Context, uris []string) error {
	if len(uris) <= 1 {
		uri := ""
		if len(uris) == 1 {
			uri = uris[0]
		}
		return c.play(ctx, uri)
	}
	deviceID := strings.TrimSpace(c.device)
	if deviceID != "" {
		if state, err := c.connectState(ctx); err == nil {
			if targetID := resolveConnectTargetDeviceID(state, deviceID); targetID != "" {
				deviceID = targetID
			}
		}
	}
	return c.playTracksViaWebAPI(ctx, uris, deviceID)
}

func (c *ConnectClient) playTracksViaWebAPI(ctx context.Context, uris []string, deviceID string) error {
	return withWebFallback(c, func(web *Client) error {
		return web.playTracks(ctx, uris, deviceID)
	})
}

func (c *ConnectClient) pause(ctx context.Context) error {
	return c.sendDirectCommand(ctx, "pause", nil)
}

func (c *ConnectClient) next(ctx context.Context) error {
	return c.sendDirectCommand(ctx, "skip_next", nil)
}

func (c *ConnectClient) previous(ctx context.Context) error {
	return c.sendDirectCommand(ctx, "skip_prev", nil)
}

func (c *ConnectClient) seek(ctx context.Context, positionMS int) error {
	if positionMS < 0 {
		positionMS = 0
	}
	return c.sendDirectCommand(ctx, "seek_to", map[string]any{
		"command": map[string]any{
			"endpoint": "seek_to",
			"value":    positionMS,
			"logging_params": map[string]any{
				"command_id": randomHex(32),
			},
		},
	})
}

func (c *ConnectClient) volume(ctx context.Context, volume int) error {
	volume = clampVolume(volume)
	return withConnectStateErr(ctx, c, func(state connectState) error {
		fromID := connectTransferSourceID(state)
		if fromID == "" || state.activeDeviceID == "" {
			return errors.New("missing device id")
		}
		url := fmt.Sprintf("%s/connect/volume/from/%s/to/%s", connectStateBase, fromID, state.activeDeviceID)
		return c.sendConnectRequest(ctx, http.MethodPut, url, map[string]any{
			"volume": int(float64(volume) / 100 * 65535),
		})
	})
}

func (c *ConnectClient) shuffle(ctx context.Context, enabled bool) error {
	return c.sendDirectCommand(ctx, "set_shuffling_context", map[string]any{
		"command": map[string]any{
			"endpoint": "set_shuffling_context",
			"value":    enabled,
			"logging_params": map[string]any{
				"command_id": randomHex(32),
			},
		},
	})
}

func (c *ConnectClient) repeat(ctx context.Context, mode string) error {
	command := map[string]any{
		"endpoint": "set_options",
		"logging_params": map[string]any{
			"command_id": randomHex(32),
		},
	}
	repeatingTrack, repeatingContext := repeatFlags(mode)
	command["repeating_track"] = repeatingTrack
	command["repeating_context"] = repeatingContext
	return c.sendDirectCommand(ctx, "set_options", map[string]any{"command": command})
}

func (c *ConnectClient) queueAdd(ctx context.Context, uri string) error {
	return c.sendDirectCommand(ctx, "add_to_queue", map[string]any{
		"command": map[string]any{
			"endpoint": "add_to_queue",
			"track": map[string]any{
				"uri": uri,
			},
			"logging_params": map[string]any{
				"command_id": randomHex(32),
			},
		},
	})
}

func (c *ConnectClient) queue(ctx context.Context) (Queue, error) {
	return withConnectState(ctx, c, func(state connectState) (Queue, error) {
		return mapQueue(state), nil
	})
}

func (c *ConnectClient) sendStateCommand(ctx context.Context, endpoint string, payload map[string]any) error {
	return withConnectStateErr(ctx, c, func(state connectState) error {
		return c.sendPlayerCommand(ctx, state, endpoint, payload)
	})
}

func (c *ConnectClient) sendDirectCommand(ctx context.Context, endpoint string, payload map[string]any) error {
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
	fromID, toID, ok := c.commandRoute()
	if !ok {
		return c.sendStateCommand(ctx, endpoint, payload)
	}
	url := fmt.Sprintf("%s/player/command/from/%s/to/%s", connectStateBase, fromID, toID)
	if err := c.sendConnectCommand(ctx, url, payload); err != nil {
		if !isRouteStaleError(err) {
			return err
		}
		return c.sendStateCommand(ctx, endpoint, payload)
	}
	return nil
}

func isRouteStaleError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.Status {
	case http.StatusBadRequest, http.StatusNotFound, http.StatusConflict, http.StatusGone:
		return true
	default:
		return false
	}
}

func withConnectState[T any](ctx context.Context, c *ConnectClient, fn func(connectState) (T, error)) (T, error) {
	state, err := c.connectState(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	return fn(state)
}

func withConnectStateErr(ctx context.Context, c *ConnectClient, fn func(connectState) error) error {
	_, err := withConnectState(ctx, c, func(state connectState) (struct{}, error) {
		return struct{}{}, fn(state)
	})
	return err
}

func connectTransferSourceID(state connectState) string {
	fromID := state.originDeviceID
	if fromID == "" {
		fromID = state.activeDeviceID
	}
	return fromID
}

func resolveConnectTargetDeviceID(state connectState, selector string) string {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return ""
	}
	for _, device := range mapDevices(state) {
		if strings.EqualFold(device.ID, selector) || strings.EqualFold(device.Name, selector) {
			return device.ID
		}
	}
	return ""
}

func playCommandPayload(uri string) map[string]any {
	command := map[string]any{
		"endpoint": "play",
		"logging_params": map[string]any{
			"command_id": randomHex(32),
		},
	}
	command["context"] = map[string]any{"uri": uri, "url": "context://" + uri}
	if !isContextURI(uri) {
		command["options"] = map[string]any{
			"skip_to": map[string]any{"track_uri": uri},
		}
	}
	return map[string]any{"command": command}
}

func clampVolume(volume int) int {
	if volume < 0 {
		return 0
	}
	if volume > 100 {
		return 100
	}
	return volume
}

func repeatFlags(mode string) (bool, bool) {
	switch strings.ToLower(mode) {
	case "track":
		return true, false
	case "context":
		return false, true
	default:
		return false, false
	}
}
