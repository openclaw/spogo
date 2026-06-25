package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

func TestWriteCompletionFish(t *testing.T) {
	parser, err := kong.New(New(), kong.Name("spogo"), kong.Vars(VersionVars()))
	if err != nil {
		t.Fatalf("parser: %v", err)
	}

	out := &bytes.Buffer{}
	if err := WriteCompletion(out, parser, "fish"); err != nil {
		t.Fatalf("completion: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"# fish completion for spogo",
		"function __spogo_command_tokens",
		"case '--config' '--profile' '--timeout' '--market' '--language' '--device' '--engine'",
		"function __spogo_seen_command_path",
		"function __spogo_needs_command",
		"complete -c spogo -f -n '__spogo_needs_command' -a 'auth'",
		"-n '__spogo_seen_command_path \\'playlist\\' \\'tracks\\'' -l limit",
		"-n '__spogo_seen_command_path \\'library\\' \\'tracks\\' \\'list\\'' -l limit",
		"-n '__spogo_seen_command_path \\'completion\\'' -a 'fish'",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("completion output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "__fish_use_subcommand") {
		t.Fatalf("completion output uses unfiltered root condition:\n%s", got)
	}
}

func TestWriteCompletionErrors(t *testing.T) {
	if err := WriteCompletion(&bytes.Buffer{}, nil, "fish"); err == nil {
		t.Fatalf("expected nil parser error")
	}
	if err := WriteCompletion(&bytes.Buffer{}, &kong.Kong{}, "bash"); err == nil {
		t.Fatalf("expected unsupported shell error")
	}
}

func TestCompletionHelpers(t *testing.T) {
	if got := fishQuote("won't"); got != "'won\\'t'" {
		t.Fatalf("fishQuote got %q", got)
	}
	if got := fishWords("playlist tracks"); got != "'playlist' 'tracks'" {
		t.Fatalf("fishWords got %q", got)
	}
	if got := fishWords(""); got != "" {
		t.Fatalf("empty fishWords got %q", got)
	}
	if got := commandPathCondition(nil); got != "" {
		t.Fatalf("empty commandPathCondition got %q", got)
	}
	if got := commandPathCondition([]string{"playlist", "tracks"}); got != "__spogo_seen_command_path 'playlist' 'tracks'" {
		t.Fatalf("commandPathCondition got %q", got)
	}
	if got := enumValues(nil); got != nil {
		t.Fatalf("nil enumValues got %#v", got)
	}
}
