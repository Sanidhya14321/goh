package skills

import (
	"context"
	"example.com/independent-harness/internal/testutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func put(t *testing.T, root, p, content string) {
	t.Helper()
	file := filepath.Join(root, p)
	if e := os.MkdirAll(filepath.Dir(file), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(file, []byte(content), 0600); e != nil {
		t.Fatal(e)
	}
}
func TestDiscoveryAndLazySkillResources(t *testing.T) {
	root := t.TempDir()
	put(t, root, ".agents/skills/mismatched/SKILL.md", "---\nname: 'Parser Helper!'\ndescription: Fixture: parser investigation\nmetadata:\n  author: fixture\n---\nPRIVATE_BODY_MARKER\nRun project tests.")
	put(t, root, ".agents/skills/mismatched/scripts/check.sh", "RESOURCE_SECRET_MARKER")
	put(t, root, ".opencode/skill/singular/SKILL.md", "---\nname: singular\n---\nSingular body")
	put(t, root, ".opencode/skills/other/SKILL.md", "---\nname: other\ndescription: >-\n  First line\n  second line\n---\nOther body")
	c, e := New(context.Background(), root, nil)
	if e != nil {
		t.Fatal(e)
	}
	if len(c.List()) != 3 {
		t.Fatalf("%+v %v", c.List(), c.Diagnostics())
	}
	description := c.Description()
	if strings.Contains(description, "PRIVATE_BODY_MARKER") || strings.Contains(description, "RESOURCE_SECRET_MARKER") {
		t.Fatal("body eagerly advertised")
	}
	if !strings.Contains(description, "Parser Helper!") {
		t.Fatal("name lost")
	}
	loaded, e := c.Load(context.Background(), "Parser Helper!")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(loaded.Content, "PRIVATE_BODY_MARKER") || strings.Contains(loaded.Content, "RESOURCE_SECRET_MARKER") || len(loaded.Files) != 1 || loaded.Files[0] != "scripts/check.sh" {
		t.Fatalf("%+v", loaded)
	}
	if !strings.HasSuffix(loaded.BaseDirectory, "mismatched") {
		t.Fatal("base directory mismatch")
	}
	list := c.List()
	list[0].Name = "changed"
	if c.List()[0].Name == "changed" {
		t.Fatal("catalog mutable alias")
	}
	if _, e = c.Resolve("unknown"); e == nil {
		t.Fatal("missing skill succeeded")
	}
}
func TestInvalidSkillsDuplicatesAndExplicitRoots(t *testing.T) {
	root, extra := t.TempDir(), t.TempDir()
	put(t, root, ".agents/skills/a/SKILL.md", "---\nname: repeated\ndescription: first\n---\nFirst")
	put(t, root, ".opencode/skills/b/SKILL.md", "---\nname: repeated\ndescription: second\n---\nSecond")
	put(t, root, ".agents/skills/number/SKILL.md", "---\nname: 7\n---\nNot valid")
	put(t, root, ".agents/skills/missing/SKILL.md", "---\ndescription: no name\n---\nNot valid")
	put(t, extra, "configured/SKILL.md", "---\nname: configured\ndescription: explicit\n---\nConfigured body")
	c, e := New(context.Background(), root, []string{extra})
	if e != nil {
		t.Fatal(e)
	}
	if len(c.List()) != 2 || len(c.Diagnostics()) < 3 {
		t.Fatalf("%+v %+v", c.List(), c.Diagnostics())
	}
	loaded, e := c.Load(context.Background(), "repeated")
	if e != nil || loaded.Content != "First" {
		t.Fatal(loaded, e)
	}
	if _, e = c.Load(context.Background(), "configured"); e != nil {
		t.Fatal(e)
	}
}
func TestSkillIdentityAndPathSafety(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	put(t, root, ".agents/skills/a/SKILL.md", "---\nname: local\n---\nBefore")
	put(t, outside, "SKILL.md", "---\nname: outside\n---\nSecret")
	if e := os.Symlink(outside, filepath.Join(root, ".agents/skills/link")); e != nil {
		t.Fatal(e)
	}
	c, e := New(context.Background(), root, nil)
	if e != nil {
		t.Fatal(e)
	}
	if len(c.List()) != 1 {
		t.Fatal("symlink discovered")
	}
	put(t, root, ".agents/skills/a/SKILL.md", "---\nname: local\n---\nAfter")
	if _, e = c.Load(context.Background(), "local"); e == nil || !strings.Contains(e.Error(), "changed") {
		t.Fatal("stale skill loaded", e)
	}
	if e = os.Remove(filepath.Join(root, ".agents/skills/a/SKILL.md")); e != nil {
		t.Fatal(e)
	}
	if e = testutil.Mkfifo(t, filepath.Join(root, ".agents/skills/a/SKILL.md")); e != nil {
		t.Fatal(e)
	}
	if _, e = c.Load(context.Background(), "local"); e == nil {
		t.Fatal("special skill loaded")
	}
}
func TestSkillBoundedResourcesAndCancellation(t *testing.T) {
	root := t.TempDir()
	put(t, root, ".agents/skills/a/SKILL.md", "---\nname: a\n---\nbody")
	for i := 0; i < 15; i++ {
		put(t, root, ".agents/skills/a/"+string(rune('a'+i))+".txt", "resource")
	}
	c, e := New(context.Background(), root, nil)
	if e != nil {
		t.Fatal(e)
	}
	loaded, e := c.Load(context.Background(), "a")
	if e != nil || len(loaded.Files) != 10 || !loaded.FilesTruncated {
		t.Fatal(loaded, e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = c.Load(ctx, "a"); e == nil {
		t.Fatal("cancelled load succeeded")
	}
	if _, e = New(ctx, root, nil); e == nil {
		t.Fatal("cancelled discovery succeeded")
	}
}
func TestFrontmatterSubset(t *testing.T) {
	for _, tc := range []struct{ raw, name, description string }{{"---\r\nname: 'a'\r\ndescription: \"quoted\"\r\n---\r\nbody", "a", "quoted"}, {"---\nname: a\ndescription: |\n  first\n  second\n---\nbody", "a", "first\nsecond"}, {"---\nname: a\ndescription: ''\n---\n", "a", ""}} {
		name, description, _, e := parse([]byte(tc.raw))
		if e != nil || name != tc.name || description != tc.description {
			t.Fatal(name, description, e)
		}
	}
	for _, bad := range []string{"no frontmatter", "---\nname: a", "---\nname: true\n---\n", "---\nname: a\ndescription: 2\n---\n", "---\nname: a\nname: b\n---\n", "---\nname: ../a\n---\n"} {
		if _, _, _, e := parse([]byte(bad)); e == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
}
func FuzzSkillFrontmatter(f *testing.F) {
	f.Add("---\nname: parser\ndescription: useful\n---\nBody")
	f.Add("---\nname: true\n---\n")
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > MaxSkillBytes {
			t.Skip()
		}
		parse([]byte(s))
	})
}
