package spotify

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) Playback(ctx context.Context) (PlaybackStatus, error) {
	var raw playbackResponse
	if err := c.get(ctx, "/me/player", nil, &raw); err != nil {
		if errors.Is(err, ErrNoContent) {
			return PlaybackStatus{IsPlaying: false}, nil
		}
		return PlaybackStatus{}, err
	}
	status := PlaybackStatus{
		IsPlaying:  raw.IsPlaying,
		ProgressMS: raw.ProgressMS,
		Shuffle:    raw.ShuffleState,
		Repeat:     raw.RepeatState,
		Device:     mapDevice(raw.Device),
	}
	if raw.Item.ID != "" {
		item := mapTrack(raw.Item)
		status.Item = &item
		if itemNeedsTrackMetadata(status.Item) {
			if full, err := c.GetTrack(ctx, status.Item.ID); err == nil {
				mergeItemMetadata(status.Item, full)
			}
		}
	}
	return status, nil
}

func (c *Client) Play(ctx context.Context, uri string) error {
	payload := map[string]any{}
	if uri != "" {
		if isContextURI(uri) {
			payload["context_uri"] = uri
		} else {
			payload["uris"] = []string{uri}
		}
	}
	return c.sendPlayback(ctx, http.MethodPut, "/me/player/play", nil, payload)
}

func (c *Client) Pause(ctx context.Context) error {
	return c.sendPlayback(ctx, http.MethodPut, "/me/player/pause", nil, nil)
}

func (c *Client) Next(ctx context.Context) error {
	return c.sendPlayback(ctx, http.MethodPost, "/me/player/next", nil, nil)
}

func (c *Client) Previous(ctx context.Context) error {
	return c.sendPlayback(ctx, http.MethodPost, "/me/player/previous", nil, nil)
}

func (c *Client) Seek(ctx context.Context, positionMS int) error {
	return c.sendPlayback(ctx, http.MethodPut, "/me/player/seek", url.Values{"position_ms": {fmt.Sprint(positionMS)}}, nil)
}

func (c *Client) Volume(ctx context.Context, volume int) error {
	return c.sendPlayback(ctx, http.MethodPut, "/me/player/volume", url.Values{"volume_percent": {fmt.Sprint(volume)}}, nil)
}

func (c *Client) Shuffle(ctx context.Context, enabled bool) error {
	return c.sendPlayback(ctx, http.MethodPut, "/me/player/shuffle", url.Values{"state": {fmt.Sprint(enabled)}}, nil)
}

func (c *Client) Repeat(ctx context.Context, mode string) error {
	return c.sendPlayback(ctx, http.MethodPut, "/me/player/repeat", url.Values{"state": {mode}}, nil)
}

func (c *Client) Devices(ctx context.Context) ([]Device, error) {
	var raw deviceResponse
	if err := c.get(ctx, "/me/player/devices", nil, &raw); err != nil {
		return nil, err
	}
	devices := make([]Device, 0, len(raw.Devices))
	for _, d := range raw.Devices {
		devices = append(devices, mapDevice(d))
	}
	return devices, nil
}

func (c *Client) Transfer(ctx context.Context, deviceID string) error {
	payload := map[string]any{"device_ids": []string{deviceID}}
	return c.put(ctx, "/me/player", payload)
}

func (c *Client) QueueAdd(ctx context.Context, uri string) error {
	return c.sendPlayback(ctx, http.MethodPost, "/me/player/queue", url.Values{"uri": {uri}}, nil)
}

func (c *Client) sendPlayback(ctx context.Context, method, path string, params url.Values, payload any) error {
	if c.device != "" {
		if params == nil {
			params = url.Values{}
		}
		params.Set("device_id", c.device)
	}
	err := c.send(ctx, method, path, params, payload, nil)
	var apiErr APIError
	if c.device == "" || !errors.As(err, &apiErr) || apiErr.Status != http.StatusNotFound {
		return err
	}
	// Device IDs are opaque. Resolve names only after a rejected selector so
	// raw IDs need neither a format heuristic nor permission to list devices.
	devices, lookupErr := c.Devices(ctx)
	if lookupErr != nil {
		return lookupErr
	}
	if device, found := FindDevice(devices, c.device); found {
		if device.ID == "" {
			return fmt.Errorf("device %q has no usable ID", c.device)
		}
		if device.ID == c.device {
			return err
		}
		params.Set("device_id", device.ID)
		return c.send(ctx, method, path, params, payload, nil)
	}
	return err
}

func (c *Client) Queue(ctx context.Context) (Queue, error) {
	var raw queueResponse
	if err := c.get(ctx, "/me/player/queue", nil, &raw); err != nil {
		if errors.Is(err, ErrNoContent) {
			return Queue{}, nil
		}
		return Queue{}, err
	}
	q := Queue{}
	if raw.CurrentlyPlaying.ID != "" {
		item := mapTrack(raw.CurrentlyPlaying)
		q.CurrentlyPlaying = &item
	}
	for _, item := range raw.Queue {
		q.Queue = append(q.Queue, mapTrack(item))
	}
	return q, nil
}
