package runtime

import "testing"

func TestReadOnlyCommand(t *testing.T) {
	for _, cmd := range []string{
		"ls", "ls -la src", "pwd; ls -d portfolio-studio 2>/dev/null; command -v node; command -v python3",
		"git status --short && git diff --stat", "git log --oneline -5", "grep -rn 'func main' .",
		"cat go.mod | head -5", "find . -name '*.go' -type f", "wc -l *.go", "node --version",
		"go version", "rg -n TODO internal", "git branch", "git branch -a", "test -f go.mod && echo yes",
	} {
		if !readOnlyCommand(cmd) {
			t.Errorf("refused read-only %q", cmd)
		}
	}
	for _, cmd := range []string{
		"rm -rf build", "ls /Applications", "cat ~/.ssh/id_rsa", "cat ../secret", "echo hi > a.txt", "ls >> log",
		"find . -delete", "find . -exec rm {} ;", "git commit -m x", "git branch -D main", "git -C /tmp status",
		"sort -o out.txt in.txt", "echo $(rm -rf .)", "echo `id`", "ls & rm x", "node -e 'process.exit()'",
		"npm install", "go build ./...", "go env -w GOFLAGS=x", "FOO=1 ls", "/bin/ls", "tee x", "cat a\nrm b",
		"sed -i s/a/b/ f", "git diff --output=x", "rg --pre sh x", "file -C -m magic", "uniq in out", "python3 x.py",
		"curl https://example.com", "ls 'unterminated",
	} {
		if readOnlyCommand(cmd) {
			t.Errorf("allowed %q", cmd)
		}
	}
}
