package cli

import (
	"os"

	"github.com/alecthomas/kong"
	kongcompletion "github.com/jotaen/kong-completion"
)

type CompletionCmd struct {
	Shell string `arg:"" enum:"bash,zsh,fish" help:"Shell to generate completions for."`
}

// RegisterCompletion handles completion requests before normal argument parsing
// and, consequently, before any Spotify configuration or authentication work.
func RegisterCompletion(parser *kong.Kong) {
	if os.Getenv("COMP_LINE") == "" {
		return
	}
	kongcompletion.Register(parser)
}

// WriteCompletion delegates the shell integration code to kong-completion while
// preserving spogo's conventional `completion <shell>` command shape.
func WriteCompletion(ctx *kong.Context, shell string) error {
	completion := kongcompletion.Completion{
		Shell: shell,
		Code:  true,
	}
	return completion.Run(ctx)
}
