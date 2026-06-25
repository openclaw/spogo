package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/kong"
)

func WriteCompletion(w io.Writer, parser *kong.Kong, shell string) error {
	if shell != "fish" {
		return fmt.Errorf("unsupported shell %q; supported shells: fish", shell)
	}
	return writeFishCompletion(w, parser)
}

func writeFishCompletion(w io.Writer, parser *kong.Kong) error {
	if parser == nil || parser.Model == nil || parser.Model.Node == nil {
		return fmt.Errorf("nil parser model")
	}

	if _, err := fmt.Fprintln(w, "# fish completion for spogo"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "complete -c spogo -f"); err != nil {
		return err
	}
	if _, err := io.WriteString(w, `function __spogo_command_tokens
    set -l tokens (commandline -opc)
    if test (count $tokens) -gt 0; and test "$tokens[1]" = spogo
        set -e tokens[1]
    end
    set -l skip_next 0
    for token in $tokens
        if test $skip_next -eq 1
            set skip_next 0
            continue
        end
        switch $token
            case '--config' '--profile' '--timeout' '--market' '--language' '--device' '--engine'
                set skip_next 1
            case '--config=*' '--profile=*' '--timeout=*' '--market=*' '--language=*' '--device=*' '--engine=*'
                continue
            case '-*'
                continue
            case '*'
                printf '%s\n' $token
        end
    end
end

function __spogo_seen_command_path
    set -l tokens (__spogo_command_tokens)
    set -l path $argv
    if test (count $tokens) -lt (count $path)
        return 1
    end
    for i in (seq (count $path))
        if test "$tokens[$i]" != "$path[$i]"
            return 1
        end
    end
    return 0
end

function __spogo_at_command_path
    set -l tokens (__spogo_command_tokens)
    set -l path $argv
    if test (count $tokens) -ne (count $path)
        return 1
    end
    __spogo_seen_command_path $path
end

function __spogo_needs_command
    test (count (__spogo_command_tokens)) -eq 0
end
`); err != nil {
		return err
	}

	return writeFishCommands(w, parser.Model.Node, nil)
}

func writeFishCommands(w io.Writer, node *kong.Node, path []string) error {
	if node == nil {
		return nil
	}

	condition := commandChildrenCondition(path)
	if node.Type == kong.ApplicationNode {
		condition = "__spogo_needs_command"
	}
	for _, child := range node.Children {
		if child.Hidden {
			continue
		}
		if _, err := fmt.Fprintf(w, "complete -c spogo -f -n %s -a %s -d %s\n",
			fishQuote(condition),
			fishQuote(commandNames(child)),
			fishQuote(child.Help),
		); err != nil {
			return err
		}
	}

	for _, flag := range node.Flags {
		if err := writeFishFlag(w, commandPathCondition(path), flag); err != nil {
			return err
		}
	}
	for _, positional := range node.Positional {
		if err := writeFishPositional(w, commandPathCondition(path), positional); err != nil {
			return err
		}
	}

	for _, child := range node.Children {
		if child.Hidden {
			continue
		}
		if err := writeFishCommands(w, child, appendCommandPath(path, child)); err != nil {
			return err
		}
	}
	return nil
}

func writeFishPositional(w io.Writer, condition string, positional *kong.Positional) error {
	if positional == nil || condition == "" {
		return nil
	}
	enum := enumValues(positional)
	if len(enum) == 0 {
		return nil
	}
	parts := []string{
		"complete",
		"-c",
		"spogo",
		"-f",
		"-n",
		fishQuote(condition),
		"-a",
		fishQuote(strings.Join(enum, " ")),
	}
	if positional.Help != "" {
		parts = append(parts, "-d", fishQuote(positional.Help))
	}
	_, err := fmt.Fprintln(w, strings.Join(parts, " "))
	return err
}

func writeFishFlag(w io.Writer, condition string, flag *kong.Flag) error {
	if flag == nil || flag.Hidden {
		return nil
	}

	parts := []string{"complete", "-c", "spogo"}
	if condition != "" {
		parts = append(parts, "-n", fishQuote(condition))
	}
	if flag.Short != 0 {
		parts = append(parts, "-s", string(flag.Short))
	}
	if flag.Name != "" {
		parts = append(parts, "-l", flag.Name)
	}
	if !flag.IsBool() {
		parts = append(parts, "-r")
	}
	if enum := enumValues(flag.Value); len(enum) > 0 {
		parts = append(parts, "-a", fishQuote(strings.Join(enum, " ")))
	}
	if flag.Help != "" {
		parts = append(parts, "-d", fishQuote(flag.Help))
	}
	_, err := fmt.Fprintln(w, strings.Join(parts, " "))
	return err
}

func commandNames(node *kong.Node) string {
	names := []string{node.Name}
	names = append(names, node.Aliases...)
	return strings.Join(names, " ")
}

func appendCommandPath(path []string, node *kong.Node) []string {
	next := make([]string, 0, len(path)+1)
	next = append(next, path...)
	next = append(next, node.Name)
	return next
}

func commandPathCondition(path []string) string {
	if len(path) == 0 {
		return ""
	}
	return "__spogo_seen_command_path " + fishWords(strings.Join(path, " "))
}

func commandChildrenCondition(path []string) string {
	if len(path) == 0 {
		return ""
	}
	return "__spogo_at_command_path " + fishWords(strings.Join(path, " "))
}

func enumValues(value *kong.Value) []string {
	if value == nil {
		return nil
	}
	values := value.EnumSlice()
	filtered := values[:0]
	for _, value := range values {
		if value != "" {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func fishWords(words string) string {
	if words == "" {
		return ""
	}
	parts := strings.Fields(words)
	for i, part := range parts {
		parts[i] = fishQuote(part)
	}
	return strings.Join(parts, " ")
}

func fishQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "\\'") + "'"
}
