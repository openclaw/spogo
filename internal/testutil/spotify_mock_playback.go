package testutil

import (
	"context"

	"github.com/steipete/spogo/internal/spotify"
)

func (m *SpotifyMock) Playback(ctx context.Context) (spotify.PlaybackStatus, error) {
	if m.PlaybackFn == nil {
		return spotify.PlaybackStatus{}, ErrNotImplemented
	}
	return m.PlaybackFn(ctx)
}

func (m *SpotifyMock) Play(ctx context.Context, uri string) error {
	if m.PlayFn == nil {
		return ErrNotImplemented
	}
	return m.PlayFn(ctx, uri)
}

func (m *SpotifyMock) PlayTracks(ctx context.Context, uris []string) error {
	if m.PlayTracksFn == nil {
		return ErrNotImplemented
	}
	return m.PlayTracksFn(ctx, uris)
}

func (m *SpotifyMock) Pause(ctx context.Context) error {
	if m.PauseFn == nil {
		return ErrNotImplemented
	}
	return m.PauseFn(ctx)
}

func (m *SpotifyMock) Next(ctx context.Context) error {
	if m.NextFn == nil {
		return ErrNotImplemented
	}
	return m.NextFn(ctx)
}

func (m *SpotifyMock) Previous(ctx context.Context) error {
	if m.PreviousFn == nil {
		return ErrNotImplemented
	}
	return m.PreviousFn(ctx)
}

func (m *SpotifyMock) Seek(ctx context.Context, positionMS int) error {
	if m.SeekFn == nil {
		return ErrNotImplemented
	}
	return m.SeekFn(ctx, positionMS)
}

func (m *SpotifyMock) Volume(ctx context.Context, volume int) error {
	if m.VolumeFn == nil {
		return ErrNotImplemented
	}
	return m.VolumeFn(ctx, volume)
}

func (m *SpotifyMock) Shuffle(ctx context.Context, enabled bool) error {
	if m.ShuffleFn == nil {
		return ErrNotImplemented
	}
	return m.ShuffleFn(ctx, enabled)
}

func (m *SpotifyMock) Repeat(ctx context.Context, mode string) error {
	if m.RepeatFn == nil {
		return ErrNotImplemented
	}
	return m.RepeatFn(ctx, mode)
}

func (m *SpotifyMock) Devices(ctx context.Context) ([]spotify.Device, error) {
	if m.DevicesFn == nil {
		return nil, ErrNotImplemented
	}
	return m.DevicesFn(ctx)
}

func (m *SpotifyMock) Transfer(ctx context.Context, deviceID string) error {
	if m.TransferFn == nil {
		return ErrNotImplemented
	}
	return m.TransferFn(ctx, deviceID)
}

func (m *SpotifyMock) QueueAdd(ctx context.Context, uri string) error {
	if m.QueueAddFn == nil {
		return ErrNotImplemented
	}
	return m.QueueAddFn(ctx, uri)
}

func (m *SpotifyMock) Queue(ctx context.Context) (spotify.Queue, error) {
	if m.QueueFn == nil {
		return spotify.Queue{}, ErrNotImplemented
	}
	return m.QueueFn(ctx)
}
