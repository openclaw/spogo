//go:build darwin
// +build darwin

package spotify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppleScriptClientLocalCommands(t *testing.T) {
	logPath := installFakeOsaScript(t)
	t.Setenv("SPOGO_OSASCRIPT_OUTPUT", "Song|||Artist|||Album|||spotify:track:1|||180000|||12.5|||playing|||42|||true|||true")

	client := &AppleScriptClient{}
	ctx := context.Background()

	calls := []func() error{
		func() error { return client.Play(ctx, "") },
		func() error { return client.Play(ctx, "spotify:track:1") },
		func() error { return client.Pause(ctx) },
		func() error { return client.Next(ctx) },
		func() error { return client.Previous(ctx) },
		func() error { return client.Seek(ctx, 12500) },
		func() error { return client.Volume(ctx, 42) },
		func() error { return client.Shuffle(ctx, true) },
		func() error { return client.Shuffle(ctx, false) },
		func() error { return client.Repeat(ctx, "track") },
		func() error { return client.Repeat(ctx, "off") },
	}
	for i, call := range calls {
		if err := call(); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}

	status, err := client.Playback(ctx)
	if err != nil {
		t.Fatalf("playback: %v", err)
	}
	if !status.IsPlaying || status.ProgressMS != 12500 || status.Device.Volume != 42 || !status.Shuffle || status.Repeat != "context" {
		t.Fatalf("unexpected status: %+v", status)
	}
	if status.Item == nil || status.Item.Name != "Song" || status.Item.URI != "spotify:track:1" {
		t.Fatalf("unexpected item: %+v", status.Item)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	log := string(logData)
	for _, want := range []string{
		"to play",
		"play track (item 1 of argv)",
		"spotify:track:1",
		"to pause",
		"to next track",
		"to previous track",
		"player position to 12",
		"sound volume to 42",
		"shuffling to true",
		"shuffling to false",
		"repeating to true",
		"repeating to false",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("script log missing %q:\n%s", want, log)
		}
	}
}

func TestAppleScriptPlaybackErrors(t *testing.T) {
	installFakeOsaScript(t)
	client := &AppleScriptClient{}
	ctx := context.Background()

	t.Setenv("SPOGO_OSASCRIPT_OUTPUT", "too few fields")
	if _, err := client.Playback(ctx); err == nil {
		t.Fatalf("expected parse error")
	}

	t.Setenv("SPOGO_OSASCRIPT_ERROR", "boom")
	if err := client.Pause(ctx); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected osascript error, got %v", err)
	}
}

func TestAppleScriptDevicesAndFallbacks(t *testing.T) {
	client, err := NewAppleScriptClient(AppleScriptOptions{})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	apple, ok := client.(*AppleScriptClient)
	if !ok {
		t.Fatalf("unexpected client type %T", client)
	}

	devices, err := apple.Devices(context.Background())
	if err != nil || len(devices) != 1 || devices[0].ID != "local" {
		t.Fatalf("devices: %+v err=%v", devices, err)
	}
	if err := apple.Transfer(context.Background(), "device"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("transfer: %v", err)
	}

	unsupported := []func() error{
		func() error { return apple.QueueAdd(context.Background(), "spotify:track:1") },
		func() error {
			_, err := apple.Queue(context.Background())
			return err
		},
		func() error {
			_, err := apple.Search(context.Background(), "track", "query", 1, 0)
			return err
		},
		func() error {
			_, err := apple.GetTrack(context.Background(), "track")
			return err
		},
		func() error {
			_, err := apple.GetAlbum(context.Background(), "album")
			return err
		},
		func() error {
			_, err := apple.GetArtist(context.Background(), "artist")
			return err
		},
		func() error {
			_, err := apple.GetPlaylist(context.Background(), "playlist")
			return err
		},
		func() error {
			_, err := apple.GetShow(context.Background(), "show")
			return err
		},
		func() error {
			_, err := apple.GetEpisode(context.Background(), "episode")
			return err
		},
		func() error {
			_, _, err := apple.LibraryTracks(context.Background(), 1, 0)
			return err
		},
		func() error {
			_, _, err := apple.LibraryAlbums(context.Background(), 1, 0)
			return err
		},
		func() error { return apple.LibraryModify(context.Background(), "tracks", []string{"id"}, "put") },
		func() error { return apple.FollowArtists(context.Background(), []string{"id"}, "put") },
		func() error {
			_, _, _, err := apple.FollowedArtists(context.Background(), 1, "")
			return err
		},
		func() error {
			_, _, err := apple.Playlists(context.Background(), 1, 0)
			return err
		},
		func() error {
			_, _, err := apple.PlaylistTracks(context.Background(), "playlist", 1, 0)
			return err
		},
		func() error {
			_, err := apple.CreatePlaylist(context.Background(), "mix", false, false)
			return err
		},
		func() error { return apple.AddTracks(context.Background(), "playlist", []string{"track"}) },
		func() error { return apple.RemoveTracks(context.Background(), "playlist", []string{"track"}) },
	}
	for i, call := range unsupported {
		if err := call(); !errors.Is(err, ErrUnsupported) {
			t.Fatalf("unsupported call %d: %v", i, err)
		}
	}

	calls := map[string]int{}
	apple.fallback = apiStub{calls: calls}
	_ = apple.QueueAdd(context.Background(), "spotify:track:1")
	_, _ = apple.Queue(context.Background())
	_, _ = apple.Search(context.Background(), "track", "query", 1, 0)
	_, _ = apple.GetTrack(context.Background(), "track")
	_, _ = apple.GetAlbum(context.Background(), "album")
	_, _ = apple.GetArtist(context.Background(), "artist")
	_, _ = apple.GetPlaylist(context.Background(), "playlist")
	_, _ = apple.GetShow(context.Background(), "show")
	_, _ = apple.GetEpisode(context.Background(), "episode")
	_, _, _ = apple.LibraryTracks(context.Background(), 1, 0)
	_, _, _ = apple.LibraryAlbums(context.Background(), 1, 0)
	_ = apple.LibraryModify(context.Background(), "tracks", []string{"id"}, "put")
	_ = apple.FollowArtists(context.Background(), []string{"id"}, "put")
	_, _, _, _ = apple.FollowedArtists(context.Background(), 1, "")
	_, _, _ = apple.Playlists(context.Background(), 1, 0)
	_, _, _ = apple.PlaylistTracks(context.Background(), "playlist", 1, 0)
	_, _ = apple.CreatePlaylist(context.Background(), "mix", false, false)
	_ = apple.AddTracks(context.Background(), "playlist", []string{"track"})
	_ = apple.RemoveTracks(context.Background(), "playlist", []string{"track"})

	for _, want := range []string{
		"QueueAdd", "Queue", "Search", "GetTrack", "GetAlbum", "GetArtist", "GetPlaylist", "GetShow", "GetEpisode",
		"LibraryTracks", "LibraryAlbums", "LibraryModify", "FollowArtists", "FollowedArtists", "Playlists", "PlaylistTracks",
		"CreatePlaylist", "AddTracks", "RemoveTracks",
	} {
		if calls[want] != 1 {
			t.Fatalf("fallback %s calls=%d", want, calls[want])
		}
	}
}

func installFakeOsaScript(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "osascript.log")
	scriptPath := filepath.Join(dir, "osascript")
	script := `#!/bin/sh
printf '%s\0' "$@" >> "$SPOGO_OSASCRIPT_LOG"
if [ -n "$SPOGO_OSASCRIPT_ERROR" ]; then
  printf '%s\n' "$SPOGO_OSASCRIPT_ERROR"
  exit 1
fi
printf '%s\n' "$SPOGO_OSASCRIPT_OUTPUT"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write osascript: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SPOGO_OSASCRIPT_LOG", logPath)
	return logPath
}

func TestAppleScriptPlayKeepsURIOutOfSource(t *testing.T) {
	logPath := installFakeOsaScript(t)
	uri := "spotify:track:quoted\"\\track\nliteral"
	if err := (&AppleScriptClient{}).Play(context.Background(), uri); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Split(strings.TrimSuffix(string(data), "\x00"), "\x00")
	if len(args) != 4 || args[0] != "-e" || args[2] != "--" || args[3] != uri {
		t.Fatalf("unexpected osascript arguments: %q", args)
	}
	if strings.Contains(args[1], uri) || !strings.Contains(args[1], "item 1 of argv") {
		t.Fatalf("URI must be passed as data, script: %q", args[1])
	}
}
