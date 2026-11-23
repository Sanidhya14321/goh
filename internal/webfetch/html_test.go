package webfetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// A repository page shaped like a code host's: global navigation, repository
// tabs, icon-only links, empty list items, hidden banners and a footer around
// a rendered README.
func TestRepositoryPageKeepsTheReadme(t *testing.T) {
	page, err := os.ReadFile("testdata/github_repo.html")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	}))
	defer server.Close()
	c := NewAllowingPrivate()
	defer c.Close()
	chrome := []string{"Skip to content", "Product", "Codespaces", "Pricing", "Sign in", "Search or jump", "Notifications", "Pull requests", "Terms", "Privacy", "githubstatus", "can’t perform", "Permalink", "TELEMETRY", "Dialog"}
	for _, format := range []string{"markdown", "text"} {
		t.Run(format, func(t *testing.T) {
			v := decoded(t, c.Fetch(context.Background(), raw(t, server.URL+"/example/fixture", format)))
			content := v["content"].(string)
			for _, want := range []string{"Title: GitHub - example/fixture: A terminal coding assistant written in Go", "Fixture", "written in Go. It edits files", "go install example.com/fixture@latest\n\nfixture --help", "Exact file tools and a patch grammar", "Durable sessions with"} {
				if !strings.Contains(content, want) {
					t.Fatalf("missing %q:\n%s", want, content)
				}
			}
			for _, unwanted := range chrome {
				if strings.Contains(content, unwanted) {
					t.Fatalf("page chrome %q kept:\n%s", unwanted, content)
				}
			}
			if strings.Contains(content, "\n\n\n") || strings.Contains(content, "[](") || strings.Contains(content, " \n") {
				t.Fatalf("blank runs, empty links or trailing spaces:\n%q", content)
			}
			for _, line := range strings.Split(content, "\n") {
				if trimmed := strings.TrimSpace(line); trimmed == "-" || trimmed == "*" || trimmed == "#" || trimmed == "##" {
					t.Fatalf("orphan marker line:\n%s", content)
				}
			}
			if format == "markdown" {
				for _, want := range []string{"# Fixture", "[Fixture](https://github.com/example/fixture) is a **terminal coding assistant**", "runs `go test`, and", "## Install\n\n```\ngo install", "fixture --help\n```", "- Exact file tools", "with *resume* and fork"} {
					if !strings.Contains(content, want) {
						t.Fatalf("markdown lost %q:\n%s", want, content)
					}
				}
			}
		})
	}
	v := decoded(t, c.Fetch(context.Background(), raw(t, server.URL, "html")))
	if v["content"] != string(page) {
		t.Fatal("raw HTML mode must return the page unchanged")
	}
}

// Without main-content markup the whole body is kept, minus page chrome.
func TestPageWithoutMainContentMarkup(t *testing.T) {
	base, _ := url.Parse("https://fixture.example/docs/")
	cases := map[string]struct {
		page           string
		want, unwanted []string
	}{
		"plain body": {
			`<html><head><title>Plain</title></head><body><header><a href="/">Home</a><nav><ul><li><a href="/a">A</a></li><li></li></ul></nav></header>
			<div><div><div><h2>Section</h2></div></div></div><p>First   paragraph.</p>


			<div></div><div><p>Second <a href="/x"></a><a href="/y">link</a>.</p></div>
			<div style="display: none">Hidden text</div><footer>Footer text</footer></body></html>`,
			[]string{"Title: Plain\n\n## Section\n\nFirst paragraph.\n\nSecond [link](https://fixture.example/y)."},
			[]string{"Home", "Hidden text", "Footer text", "[](", "\n\n\n"},
		},
		"two articles are ambiguous": {
			`<body><nav>Menu</nav><article><header><h1>One</h1></header><p>Alpha</p></article><article><h1>Two</h1><p>Beta</p></article></body>`,
			[]string{"# One", "Alpha", "# Two", "Beta"},
			[]string{"Menu"},
		},
		"main without a readme": {
			`<body><div>Banner text</div><main><h1>Docs</h1><aside role="navigation">Contents</aside><p>Body</p></main><div>After</div></body>`,
			[]string{"# Docs\n\nBody"},
			[]string{"Banner text", "Contents", "After"},
		},
		"empty main falls back to the page": {
			`<body><main></main><p>Server text</p></body>`,
			[]string{"Server text"},
			nil,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got, _, err := renderHTML(context.Background(), []byte(c.page), "markdown", base)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range c.want {
				if !strings.Contains(got, want) {
					t.Fatalf("missing %q:\n%q", want, got)
				}
			}
			for _, unwanted := range c.unwanted {
				if strings.Contains(got, unwanted) {
					t.Fatalf("kept %q:\n%q", unwanted, got)
				}
			}
		})
	}
}

// Blank lines are normalized only outside preformatted text.
func TestPreformattedBlankLinesSurvive(t *testing.T) {
	base, _ := url.Parse("https://fixture.example/")
	page := "<main><p>a</p><div></div><div></div><p>b</p><pre>x\n\n\n\ny  </pre><p>c</p></main>"
	for format, want := range map[string]string{
		"markdown": "a\n\nb\n\n```\nx\n\n\n\ny  \n```\n\nc",
		"text":     "a\nb\nx\n\n\n\ny  \nc",
	} {
		got, _, err := renderHTML(context.Background(), []byte(page), format, base)
		if err != nil || got != want {
			t.Fatalf("%s: %q, %v", format, got, err)
		}
	}
}

// Candidates are chosen with the visibility rules used for rendering, and a
// main element keeps its own header and notes.
func TestMainContentChoiceUsesVisibility(t *testing.T) {
	base, _ := url.Parse("https://fixture.example/")
	for name, c := range map[string]struct {
		page           string
		want, unwanted []string
	}{
		"hidden main is not a candidate": {
			`<body><div hidden><main><p>Stale copy</p></main></div><header><a href="/">Site</a></header><main><header><h1>Guide</h1><p>Updated today</p></header><aside class="note"><p>Note: keep this</p></aside><p>Body text</p><footer>Page footnote</footer></main><footer>Site footer</footer></body>`,
			[]string{"# Guide", "Updated today", "Note: keep this", "Body text", "Page footnote"},
			[]string{"Stale copy", "Site footer", "Site\n"},
		},
		"readme in a template or nav is not a candidate": {
			`<body><template><article class="markdown-body"><p>Template</p></article></template><nav><article class="markdown-body"><p>Menu card</p></article></nav><article class="markdown-body"><h1>Real</h1><p>Readme</p></article><p>Outside</p></body>`,
			[]string{"# Real\n\nReadme"},
			[]string{"Template", "Menu card", "Outside"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, _, err := renderHTML(context.Background(), []byte(c.page), "markdown", base)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range c.want {
				if !strings.Contains(got, want) {
					t.Fatalf("missing %q:\n%q", want, got)
				}
			}
			for _, unwanted := range c.unwanted {
				if strings.Contains(got+"\n", unwanted) {
					t.Fatalf("kept %q:\n%q", unwanted, got)
				}
			}
		})
	}
}
