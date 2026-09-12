package cli

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/steipete/spogo/internal/app"
	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
	"github.com/steipete/spogo/internal/testutil"
)

func TestPlaylistAddCmd(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	called := false
	mock := &testutil.SpotifyMock{
		AddTracksFn: func(ctx context.Context, playlistID string, uris []string) error {
			called = true
			if playlistID != "p1" {
				t.Fatalf("playlist id %s", playlistID)
			}
			if len(uris) != 1 || uris[0] != "spotify:track:t1" {
				t.Fatalf("uris: %#v", uris)
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistAddCmd{Playlist: "spotify:playlist:p1", Tracks: []string{"spotify:track:t1"}}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Fatalf("expected call")
	}
}

func TestPlaylistAddCmdError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		AddTracksFn: func(ctx context.Context, playlistID string, uris []string) error {
			return errors.New("boom")
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistAddCmd{Playlist: "spotify:playlist:p1", Tracks: []string{"spotify:track:t1"}}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestPlaylistCreateCmd(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		CreatePlaylistFn: func(ctx context.Context, name string, public, collaborative bool) (spotify.Item, error) {
			return spotify.Item{ID: "p1", Name: name, Type: "playlist"}, nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistCreateCmd{Name: "Road Trip"}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.String() == "" {
		t.Fatalf("expected output")
	}
}

func TestPlaylistTracksCmd(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		PlaylistTracksFn: func(ctx context.Context, id string, limit, offset int) ([]spotify.Item, int, error) {
			return []spotify.Item{{ID: "t1", Name: "Track", Type: "track"}}, 1, nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistTracksCmd{Playlist: "spotify:playlist:p1", Limit: 1}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.String() == "" {
		t.Fatalf("expected output")
	}
}

func TestPlaylistRemoveCmd(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		RemoveTracksFn: func(ctx context.Context, playlistID string, uris []string) error {
			if playlistID != "p1" {
				t.Fatalf("playlist %s", playlistID)
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistRemoveCmd{Playlist: "spotify:playlist:p1", Tracks: []string{"spotify:track:t1"}}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestPlaylistMembershipCommands(t *testing.T) {
	for _, format := range []output.Format{output.FormatPlain, output.FormatJSON, output.FormatHuman} {
		t.Run(string(format), func(t *testing.T) {
			following := false
			requests, lookups := 0, 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/playlists/p1" {
					lookups++
					_, _ = w.Write([]byte(`{"id":"p1","name":"Road Trip","type":"playlist"}`))
					return
				}
				requests++
				if r.URL.Query().Encode() != "uris=spotify%3Aplaylist%3Ap1" {
					t.Errorf("unexpected query: %s", r.URL.RawQuery)
				}
				switch {
				case r.Method == http.MethodPut && r.URL.Path == "/me/library":
					following = true
				case r.Method == http.MethodDelete && r.URL.Path == "/me/library":
					following = false
				case r.Method == http.MethodGet && r.URL.Path == "/me/library/contains":
					_ = json.NewEncoder(w).Encode([]bool{following})
					return
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()
			client, err := spotify.NewClient(spotify.Options{TokenProvider: playlistTokenProvider{}, BaseURL: srv.URL})
			if err != nil {
				t.Fatal(err)
			}
			ctx, out, _ := testutil.NewTestContext(t, format)
			ctx.SetSpotify(client)
			steps := []struct {
				cmd                interface{ Run(*app.Context) error }
				plain, human, json string
			}{
				{&PlaylistFollowingCmd{Playlist: "p1"}, "false\n", "Following: false\n", `{"following":false}`},
				{&PlaylistFollowCmd{Playlist: "https://open.spotify.com/playlist/p1"}, "ok\n", "Followed Road Trip\n", `{"status":"ok","id":"p1"}`},
				{&PlaylistFollowingCmd{Playlist: "spotify:playlist:p1"}, "true\n", "Following: true\n", `{"following":true}`},
				{&PlaylistUnfollowCmd{Playlist: "spotify:playlist:p1"}, "ok\n", "Unfollowed Road Trip\n", `{"status":"ok","id":"p1"}`},
				{&PlaylistFollowingCmd{Playlist: "p1"}, "false\n", "Following: false\n", `{"following":false}`},
			}
			for _, step := range steps {
				out.Reset()
				if err := step.cmd.Run(ctx); err != nil {
					t.Fatal(err)
				}
				switch format {
				case output.FormatPlain:
					if out.String() != step.plain {
						t.Fatalf("output %q, want %q", out.String(), step.plain)
					}
				case output.FormatHuman:
					if out.String() != step.human {
						t.Fatalf("output %q, want %q", out.String(), step.human)
					}
				case output.FormatJSON:
					var got, want map[string]any
					if err := json.Unmarshal(out.Bytes(), &got); err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal([]byte(step.json), &want); err != nil {
						t.Fatal(err)
					}
					if len(got) != len(want) {
						t.Fatalf("payload: %#v", got)
					}
					for key, value := range want {
						if got[key] != value {
							t.Fatalf("payload: %#v, want %#v", got, want)
						}
					}
				}
			}
			wantLookups := 0
			if format == output.FormatHuman {
				wantLookups = 2
			}
			if requests != 5 || lookups != wantLookups {
				t.Fatalf("requests=%d lookups=%d", requests, lookups)
			}
		})
	}
}

type playlistTokenProvider struct{}

func (playlistTokenProvider) Token(context.Context) (spotify.Token, error) {
	return spotify.Token{AccessToken: "test-token"}, nil
}

func TestPlaylistMembershipErrors(t *testing.T) {
	for _, makeCmd := range []func(string) interface{ Run(*app.Context) error }{
		func(id string) interface{ Run(*app.Context) error } { return &PlaylistFollowCmd{Playlist: id} },
		func(id string) interface{ Run(*app.Context) error } { return &PlaylistUnfollowCmd{Playlist: id} },
		func(id string) interface{ Run(*app.Context) error } { return &PlaylistFollowingCmd{Playlist: id} },
	} {
		ctx, out, _ := testutil.NewTestContext(t, output.FormatJSON)
		ctx.SetSpotify(&testutil.SpotifyMock{})
		if err := makeCmd("spotify:track:t1").Run(ctx); err == nil {
			t.Fatal("expected wrong-type error")
		}
		if err := makeCmd("p1").Run(ctx); !errors.Is(err, testutil.ErrNotImplemented) {
			t.Fatalf("error: %v", err)
		}
		if out.Len() != 0 {
			t.Fatalf("success output on failure: %s", out)
		}
		ctx, _, _ = testutil.NewTestContext(t, output.FormatPlain)
		ctx.Profile.Engine = "invalid"
		if err := makeCmd("p1").Run(ctx); err == nil {
			t.Fatal("expected client construction error")
		}
	}
}

func TestPlaylistMembershipNameLookupFailure(t *testing.T) {
	for _, lookupErr := range []error{nil, errors.New("lookup failed")} {
		ctx, out, _ := testutil.NewTestContext(t, output.FormatHuman)
		ctx.SetSpotify(&testutil.SpotifyMock{
			FollowPlaylistFn:   func(context.Context, string) error { return nil },
			UnfollowPlaylistFn: func(context.Context, string) error { return nil },
			GetPlaylistFn:      func(context.Context, string) (spotify.Item, error) { return spotify.Item{}, lookupErr },
		})
		if err := (&PlaylistFollowCmd{Playlist: "p1"}).Run(ctx); err != nil {
			t.Fatal(err)
		}
		if err := (&PlaylistUnfollowCmd{Playlist: "p1"}).Run(ctx); err != nil {
			t.Fatal(err)
		}
		if out.String() != "Followed p1\nUnfollowed p1\n" {
			t.Fatalf("output: %q", out.String())
		}
	}
}

func TestPlaylistFollowRejectsPublicFlag(t *testing.T) {
	parser, err := kong.New(New(), kong.Vars(VersionVars()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.Parse([]string{"playlist", "follow", "p1", "--public"}); err == nil || !strings.Contains(err.Error(), "--public") {
		t.Fatalf("expected unknown --public flag, got %v", err)
	}
}
