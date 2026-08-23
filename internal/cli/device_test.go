package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
	"github.com/steipete/spogo/internal/testutil"
)

func TestDeviceSetCmd(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	called := false
	mock := &testutil.SpotifyMock{
		DevicesFn: func(ctx context.Context) ([]spotify.Device, error) {
			return []spotify.Device{{ID: "d1", Name: "Desk"}}, nil
		},
		TransferFn: func(ctx context.Context, deviceID string) error {
			called = true
			if deviceID != "d1" {
				t.Fatalf("device id %s", deviceID)
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := DeviceSetCmd{Device: "Desk"}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Fatalf("expected transfer")
	}
}

func TestDeviceSetCmdTransferError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		DevicesFn: func(ctx context.Context) ([]spotify.Device, error) {
			return []spotify.Device{{ID: "d1", Name: "Desk"}}, nil
		},
		TransferFn: func(ctx context.Context, deviceID string) error {
			return errors.New("boom")
		},
	}
	ctx.SetSpotify(mock)
	cmd := DeviceSetCmd{Device: "Desk"}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDeviceListCmd(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		DevicesFn: func(ctx context.Context) ([]spotify.Device, error) {
			return []spotify.Device{{ID: "d1", Name: "Desk", Active: true}}, nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := DeviceListCmd{}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.String() == "" {
		t.Fatalf("expected output")
	}
}

func TestDeviceListHumanOmitsTrailingSpace(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatHuman)
	ctx.SetSpotify(&testutil.SpotifyMock{
		DevicesFn: func(context.Context) ([]spotify.Device, error) {
			return []spotify.Device{
				{ID: "d1", Name: "Schlafzimmer", Type: "SPEAKER"},
				{ID: "d2", Name: "Desk", Type: "COMPUTER", Active: true},
			}, nil
		},
	})
	if err := (&DeviceListCmd{}).Run(ctx); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "Schlafzimmer (SPEAKER)\nDesk (COMPUTER) (active)\n"; got != want {
		t.Fatalf("output=%q want=%q", got, want)
	}
}

func TestDeviceListCmdError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		DevicesFn: func(ctx context.Context) ([]spotify.Device, error) {
			return nil, errors.New("boom")
		},
	}
	ctx.SetSpotify(mock)
	cmd := DeviceListCmd{}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestActiveMarker(t *testing.T) {
	if activeMarker(false) != "" {
		t.Fatalf("expected empty")
	}
	if activeMarker(true) == "" {
		t.Fatalf("expected marker")
	}
}
