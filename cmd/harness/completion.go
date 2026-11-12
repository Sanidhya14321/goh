package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// subcommands are completed after their top-level command.
var subcommands = map[string][]string{
	"sessions":   {"list", "preview", "show", "export", "import", "fork", "salvage", "rename", "delete", "restore", "prune"},
	"providers":  {"list", "use", "remove"},
	"auth":       {"login", "switch", "status", "models", "logout", "list", "store"},
	"mcp":        {"list"},
	"memory":     {"add", "list", "recall", "delete"},
	"completion": {"bash", "zsh", "fish"},
	"worktree":   {"list"},
	"snapshots":  {"list", "diff", "restore", "undo"},
}

// sessionIDArgs take a session ID next; IDs come from
// `goh sessions list --format ids`, which reads only local journals.
var sessionIDArgs = []string{"preview", "show", "export", "fork", "salvage", "rename", "delete", "restore"}

const idsCommand = "goh sessions list --format ids 2>/dev/null"

// writeCompletion prints a shell completion script. Nothing is installed;
// the user sources it.
func writeCompletion(w io.Writer, shell string) error {
	names := make([]string, 0, len(commandHelp))
	for _, c := range commandHelp {
		names = append(names, c[0])
	}
	words := strings.Join(names, " ")
	parents := make([]string, 0, len(subcommands))
	for k := range subcommands {
		parents = append(parents, k)
	}
	sort.Strings(parents)
	idSubs := strings.Join(sessionIDArgs, "|")
	switch shell {
	case "bash":
		fmt.Fprintf(w, "# goh bash completion: eval \"$(goh completion bash)\"\n_goh() {\n  local cur=${COMP_WORDS[COMP_CWORD]} cmd=${COMP_WORDS[1]}\n  if [ \"$COMP_CWORD\" -eq 1 ]; then\n    COMPREPLY=($(compgen -W %q -- \"$cur\")); return\n  fi\n  case \"$cmd\" in\n", words)
		fmt.Fprintf(w, "    resume|compact)\n      if [ \"$COMP_CWORD\" -eq 2 ]; then COMPREPLY=($(compgen -W \"$(%s)\" -- \"$cur\")); return; fi ;;\n", idsCommand)
		for _, p := range parents {
			fmt.Fprintf(w, "    %s)\n      if [ \"$COMP_CWORD\" -eq 2 ]; then COMPREPLY=($(compgen -W %q -- \"$cur\")); return; fi\n", p, strings.Join(subcommands[p], " "))
			if p == "sessions" {
				fmt.Fprintf(w, "      case \"${COMP_WORDS[2]}\" in %s) if [ \"$COMP_CWORD\" -eq 3 ]; then COMPREPLY=($(compgen -W \"$(%s)\" -- \"$cur\")); return; fi ;; esac\n", idSubs, idsCommand)
			}
			fmt.Fprintln(w, "      ;;")
		}
		fmt.Fprintln(w, "  esac\n  COMPREPLY=($(compgen -f -- \"$cur\"))\n}\ncomplete -o filenames -F _goh goh")
	case "zsh":
		fmt.Fprintln(w, "#compdef goh\n# goh zsh completion: eval \"$(goh completion zsh)\"\n_goh() {\n  local -a commands\n  commands=(")
		for _, c := range commandHelp {
			fmt.Fprintf(w, "    %s\n", zshQuote(c[0]+":"+c[1]))
		}
		fmt.Fprintln(w, "  )\n  if (( CURRENT == 2 )); then\n    _describe command commands; return\n  fi\n  case $words[2] in")
		fmt.Fprintf(w, "    resume|compact) (( CURRENT == 3 )) && { compadd -- ${(f)\"$(%s)\"}; return } ;;\n", idsCommand)
		for _, p := range parents {
			fmt.Fprintf(w, "    %s)\n      (( CURRENT == 3 )) && { compadd -- %s; return }\n", p, strings.Join(subcommands[p], " "))
			if p == "sessions" {
				fmt.Fprintf(w, "      [[ $words[3] == (%s) ]] && (( CURRENT == 4 )) && { compadd -- ${(f)\"$(%s)\"}; return }\n", idSubs, idsCommand)
			}
			fmt.Fprintln(w, "      ;;")
		}
		fmt.Fprintln(w, "  esac\n  _files\n}\ncompdef _goh goh")
	case "fish":
		fmt.Fprintln(w, "# goh fish completion: goh completion fish | source")
		for _, c := range commandHelp {
			fmt.Fprintf(w, "complete -c goh -f -n __fish_use_subcommand -a %s -d %s\n", c[0], fishQuote(c[1]))
		}
		fmt.Fprintf(w, "complete -c goh -f -n '__fish_seen_subcommand_from resume compact' -a '(%s)'\n", idsCommand)
		for _, p := range parents {
			fmt.Fprintf(w, "complete -c goh -f -n '__fish_seen_subcommand_from %s' -a %s\n", p, fishQuote(strings.Join(subcommands[p], " ")))
		}
		fmt.Fprintf(w, "complete -c goh -f -n '__fish_seen_subcommand_from %s' -a '(%s)'\n", strings.Join(sessionIDArgs, " "), idsCommand)
	default:
		return fmt.Errorf("usage: goh completion bash|zsh|fish")
	}
	return nil
}

func zshQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
func fishQuote(s string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, "'", `\'`).Replace(s) + "'"
}
