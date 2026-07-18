package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/alecthomas/kong"
	"github.com/steipete/spogo/internal/app"
	"github.com/steipete/spogo/internal/output"
)

func New() *CLI {
	return &CLI{}
}

type CLI struct {
	Globals Globals `kong:"embed"`

	Auth       AuthCmd       `kong:"cmd,help='Authentication and cookies.'"`
	Completion CompletionCmd `kong:"cmd,help='Generate shell completions.'"`
	Search     SearchCmd     `kong:"cmd,help='Search Spotify.'"`
	Track      TrackCmd      `kong:"cmd,help='Track operations.'"`
	Album      AlbumCmd      `kong:"cmd,help='Album operations.'"`
	Artist     ArtistCmd     `kong:"cmd,help='Artist operations.'"`
	Playlist   PlaylistCmd   `kong:"cmd,help='Playlist operations.'"`
	Show       ShowCmd       `kong:"cmd,help='Show operations.'"`
	Episode    EpisodeCmd    `kong:"cmd,help='Episode operations.'"`

	Play    PlayCmd    `kong:"cmd,help='Start playback.'"`
	Pause   PauseCmd   `kong:"cmd,help='Pause playback.'"`
	Next    NextCmd    `kong:"cmd,help='Skip to next.'"`
	Prev    PrevCmd    `kong:"cmd,help='Skip to previous.'"`
	Seek    SeekCmd    `kong:"cmd,help='Seek within track.'"`
	Volume  VolumeCmd  `kong:"cmd,help='Set volume.'"`
	Shuffle ShuffleCmd `kong:"cmd,help='Toggle shuffle.'"`
	Repeat  RepeatCmd  `kong:"cmd,help='Set repeat mode.'"`
	Status  StatusCmd  `kong:"cmd,help='Playback status.'"`

	Queue   QueueCmd   `kong:"cmd,help='Queue operations.'"`
	Library LibraryCmd `kong:"cmd,help='Library operations.'"`
	Device  DeviceCmd  `kong:"cmd,help='Playback devices.'"`
	User    UserCmd    `kong:"cmd,help='User listening data.'"`
}

type Globals struct {
	Config   string           `help:"Config file path." env:"SPOGO_CONFIG"`
	Profile  string           `help:"Profile name." env:"SPOGO_PROFILE"`
	Timeout  time.Duration    `help:"HTTP timeout." env:"SPOGO_TIMEOUT" default:"10s"`
	Market   string           `help:"Market country code." env:"SPOGO_MARKET"`
	Language string           `help:"Language/locale." env:"SPOGO_LANGUAGE"`
	Device   string           `help:"Device name or id." env:"SPOGO_DEVICE"`
	Engine   string           `help:"Engine (auto|web|connect|applescript)." env:"SPOGO_ENGINE"`
	JSON     bool             `help:"JSON output." env:"SPOGO_JSON"`
	Plain    bool             `help:"Plain output." env:"SPOGO_PLAIN"`
	NoColor  bool             `help:"Disable color output." env:"SPOGO_NO_COLOR"`
	Quiet    bool             `short:"q" help:"Quiet output." env:"SPOGO_QUIET"`
	Verbose  bool             `short:"v" help:"Verbose output." env:"SPOGO_VERBOSE"`
	Debug    bool             `short:"d" help:"Debug output." env:"SPOGO_DEBUG"`
	NoInput  bool             `help:"Disable prompts." env:"SPOGO_NO_INPUT"`
	Version  kong.VersionFlag `help:"Print version."`
}

func (g Globals) Settings() (app.Settings, error) {
	format, err := outputFormat(g.JSON, g.Plain)
	if err != nil {
		return app.Settings{}, err
	}
	return app.Settings{
		ConfigPath: g.Config,
		Profile:    g.Profile,
		Timeout:    g.Timeout,
		Market:     g.Market,
		Language:   g.Language,
		Device:     g.Device,
		Engine:     g.Engine,
		Format:     format,
		NoColor:    g.NoColor,
		Quiet:      g.Quiet,
		Verbose:    g.Verbose,
		Debug:      g.Debug,
		NoInput:    g.NoInput,
	}, nil
}

func outputFormat(jsonFlag, plainFlag bool) (output.Format, error) {
	if jsonFlag && plainFlag {
		return "", errors.New("--json and --plain are mutually exclusive")
	}
	if jsonFlag {
		return output.FormatJSON, nil
	}
	if plainFlag {
		return output.FormatPlain, nil
	}
	return output.FormatHuman, nil
}

func VersionVars() map[string]string {
	return map[string]string{
		"version": currentVersion(),
	}
}

func Usage() string {
	return fmt.Sprintf("spogo %s", currentVersion())
}
