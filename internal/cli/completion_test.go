package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

func TestWriteCompletion(t *testing.T) {
	tests := []struct {
		shell string
		want  []string
	}{
		{
			shell: "bash",
			want:  []string{"complete", "-C", "spogo"},
		},
		{
			shell: "zsh",
			want:  []string{"#compdef spogo", "bashcompinit", "complete", "-C", "spogo"},
		},
		{
			shell: "fish",
			want:  []string{"function __complete_spogo", "COMP_LINE", "complete -f -c spogo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			out := &bytes.Buffer{}
			exitCode := -1
			parser, err := kong.New(New(),
				kong.Name("spogo"),
				kong.Writers(out, &bytes.Buffer{}),
				kong.Vars(VersionVars()),
				kong.Exit(func(code int) { exitCode = code }),
			)
			if err != nil {
				t.Fatalf("parser: %v", err)
			}
			ctx, err := parser.Parse([]string{"completion", tt.shell})
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if err := WriteCompletion(ctx, tt.shell); err != nil {
				t.Fatalf("completion: %v", err)
			}
			if exitCode != 0 {
				t.Fatalf("exit code: got %d, want 0", exitCode)
			}
			for _, want := range tt.want {
				if !strings.Contains(out.String(), want) {
					t.Errorf("completion output missing %q:\n%s", want, out.String())
				}
			}
		})
	}
}

func TestRegisterCompletionPredictsFromKongModel(t *testing.T) {
	t.Setenv("COMP_LINE", "spogo playlist ")
	t.Setenv("COMP_POINT", "15")

	out := &bytes.Buffer{}
	exitCode := -1
	parser, err := kong.New(New(),
		kong.Name("spogo"),
		kong.Writers(out, &bytes.Buffer{}),
		kong.Vars(VersionVars()),
		kong.Exit(func(code int) { exitCode = code }),
	)
	if err != nil {
		t.Fatalf("parser: %v", err)
	}

	RegisterCompletion(parser)
	if exitCode != 0 {
		t.Fatalf("exit code: got %d, want 0", exitCode)
	}
	for _, want := range []string{"tracks", "create", "remove"} {
		if !strings.Contains(out.String(), want+"\n") {
			t.Errorf("completion predictions missing %q:\n%s", want, out.String())
		}
	}
}
