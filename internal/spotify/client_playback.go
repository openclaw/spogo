package spotify

import (
	"context"
	"errors"
	"fmt"
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
	return c.put(ctx, "/me/player/play", payload)
}

func (c *Client) Pause(ctx context.Context) error {
	return c.put(ctx, "/me/player/pause", nil)
}

func (c *Client) Next(ctx context.Context) error {
	return c.post(ctx, "/me/player/next", nil)
}

func (c *Client) Previous(ctx context.Context) error {
	return c.post(ctx, "/me/player/previous", nil)
}

func (c *Client) Seek(ctx context.Context, positionMS int) error {
	params := url.Values{}
	params.Set("position_ms", fmt.Sprint(positionMS))
	return c.putParams(ctx, "/me/player/seek", params)
}

func (c *Client) Volume(ctx context.Context, volume int) error {
	params := url.Values{}
	params.Set("volume_percent", fmt.Sprint(volume))
	return c.putParams(ctx, "/me/player/volume", params)
}

func (c *Client) Shuffle(ctx context.Context, enabled bool) error {
	params := url.Values{}
	params.Set("state", fmt.Sprint(enabled))
	return c.putParams(ctx, "/me/player/shuffle", params)
}

func (c *Client) Repeat(ctx context.Context, mode string) error {
	params := url.Values{}
	params.Set("state", mode)
	return c.putParams(ctx, "/me/player/repeat", params)
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
	params := url.Values{}
	params.Set("uri", uri)
	return c.postParams(ctx, "/me/player/queue", params)
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
