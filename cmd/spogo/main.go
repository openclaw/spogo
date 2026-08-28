package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/steipete/spogo/internal/app"
	"github.com/steipete/spogo/internal/cli"
)

var exitFunc = os.Exit

func main() {
	exitFunc(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, out io.Writer, errOut io.Writer) int {
	command := cli.New()
	exitCode := -1
	parser, err := kong.New(
		command,
		kong.Name("spogo"),
		kong.Description("Spotify power CLI using browser cookies or official OAuth."),
		kong.UsageOnError(),
		kong.Writers(out, errOut),
		kong.Vars(cli.VersionVars()),
		kong.Exit(func(code int) {
			exitCode = code
		}),
	)
	if err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return 2
	}
	cli.RegisterCompletion(parser)
	if exitCode >= 0 {
		return exitCode
	}
	args = normalizeArgs(args)
	kctx, err := parser.Parse(args)
	if exitCode >= 0 {
		return exitCode
	}
	if err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return 2
	}
	if command.Completion.Shell != "" {
		if err := cli.WriteCompletion(kctx, command.Completion.Shell); err != nil {
			_, _ = fmt.Fprintln(errOut, err)
			return 2
		}
		return 0
	}
	settings, err := command.Globals.Settings()
	if err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return 2
	}
	ctx, err := app.NewContext(settings)
	if err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return 1
	}
	ctx.SetCommandContext(context.Background())
	if err := ctx.ValidateProfile(); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return 2
	}
	if err := kctx.Run(ctx); err != nil {
		ctx.Output.Errorf("%v", err)
		return app.ExitCode(err)
	}
	return 0
}

func normalizeArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	front := make([]string, 0, 1)
	rest := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--no-input" {
			front = append(front, arg)
			continue
		}
		rest = append(rest, arg)
	}
	if len(front) == 0 {
		return args
	}
	normalized := make([]string, 0, len(args))
	normalized = append(normalized, front...)
	normalized = append(normalized, rest...)
	return normalized
}
