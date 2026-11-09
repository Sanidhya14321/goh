package main

import (
	"fmt"
	"io"
	"strings"
)

// writeCompletion prints a shell completion script for the top-level
// commands in commandHelp. Nothing is installed; the user sources it.
func writeCompletion(w io.Writer, shell string) error {
	names := make([]string, 0, len(commandHelp)+1)
	for _, c := range commandHelp {
		names = append(names, c[0])
	}
	words := strings.Join(names, " ")
	switch shell {
	case "bash":
		fmt.Fprintf(w, `# goh bash completion: eval "$(goh completion bash)"
_goh() {
  local cur=${COMP_WORDS[COMP_CWORD]}
  if [ "$COMP_CWORD" -eq 1 ]; then
    COMPREPLY=($(compgen -W %q -- "$cur"))
  elif [ "$COMP_CWORD" -eq 2 ] && [ "${COMP_WORDS[1]}" = completion ]; then
    COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur"))
  else
    COMPREPLY=($(compgen -f -- "$cur"))
  fi
}
complete -o filenames -F _goh goh
`, words)
	case "zsh":
		fmt.Fprintln(w, "#compdef goh\n# goh zsh completion: eval \"$(goh completion zsh)\"\n_goh() {\n  local -a commands\n  commands=(")
		for _, c := range commandHelp {
			fmt.Fprintf(w, "    %s\n", zshQuote(c[0]+":"+c[1]))
		}
		fmt.Fprintln(w, "  )\n  if (( CURRENT == 2 )); then\n    _describe command commands\n  elif [[ $words[2] == completion ]]; then\n    _values shell bash zsh fish\n  else\n    _files\n  fi\n}\ncompdef _goh goh")
	case "fish":
		fmt.Fprintln(w, "# goh fish completion: goh completion fish | source")
		for _, c := range commandHelp {
			fmt.Fprintf(w, "complete -c goh -n __fish_use_subcommand -a %s -d %s\n", c[0], fishQuote(c[1]))
		}
		fmt.Fprintln(w, "complete -c goh -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish'")
	default:
		return fmt.Errorf("usage: goh completion bash|zsh|fish")
	}
	return nil
}

func zshQuote(s string) string  { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
func fishQuote(s string) string { return "'" + strings.NewReplacer(`\`, `\\`, "'", `\'`).Replace(s) + "'" }
