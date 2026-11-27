package webfetch

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// A selector is a small CSS subset: comma-separated groups of compound
// selectors joined by descendant (space) or child (>) combinators; a compound
// is a tag or *, then any #id, .class, [attr] or [attr=value].
type selector [][]step

type step struct {
	child    bool // joined to the previous step by >
	tag, id  string
	classes  []string
	attrs    [][2]string // name, value; value "\x00" means any value
	anyValue []bool
}

const (
	maxSelector = 512
	maxMatches  = 50
	maxLinks    = 100
	linkBudget  = 16 << 10
)

var errSelector = errors.New("webfetch selector supports tag, *, #id, .class, [attr], [attr=value], descendant and > combinators, and commas")

func parseSelector(s string) (selector, error) {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > maxSelector {
		return nil, errSelector
	}
	var sel selector
	for _, group := range strings.Split(s, ",") {
		var steps []step
		child := false
		for _, tok := range tokenizeSelector(group) {
			if tok == ">" {
				if child || len(steps) == 0 {
					return nil, errSelector
				}
				child = true
				continue
			}
			st, err := parseCompound(tok)
			if err != nil {
				return nil, err
			}
			st.child = child
			child = false
			steps = append(steps, st)
		}
		if len(steps) == 0 || child || len(steps) > 16 {
			return nil, errSelector
		}
		sel = append(sel, steps)
	}
	if len(sel) > 16 {
		return nil, errSelector
	}
	return sel, nil
}

// tokenizeSelector splits on whitespace and >, keeping [..] intact.
func tokenizeSelector(s string) []string {
	var out []string
	var cur strings.Builder
	depth := 0
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		switch {
		case r == '[':
			depth++
			cur.WriteRune(r)
		case r == ']':
			depth--
			cur.WriteRune(r)
		case depth > 0:
			cur.WriteRune(r)
		case r == '>':
			flush()
			out = append(out, ">")
		case r == ' ' || r == '\t' || r == '\n':
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

func identChar(c byte) bool {
	return c == '-' || c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}

func readIdent(s string) (string, string) {
	i := 0
	for i < len(s) && identChar(s[i]) {
		i++
	}
	return s[:i], s[i:]
}

func parseCompound(s string) (step, error) {
	var st step
	if strings.HasPrefix(s, "*") {
		s = s[1:]
	} else {
		st.tag, s = readIdent(s)
		st.tag = strings.ToLower(st.tag)
	}
	for s != "" {
		var name string
		switch s[0] {
		case '#':
			if name, s = readIdent(s[1:]); name == "" || st.id != "" {
				return st, errSelector
			}
			st.id = name
		case '.':
			if name, s = readIdent(s[1:]); name == "" {
				return st, errSelector
			}
			st.classes = append(st.classes, name)
		case '[':
			end := strings.IndexByte(s, ']')
			if end < 0 {
				return st, errSelector
			}
			body := s[1:end]
			s = s[end+1:]
			key, value, hasValue := strings.Cut(body, "=")
			key = strings.ToLower(strings.TrimSpace(key))
			if k, rest := readIdent(key); k == "" || rest != "" {
				return st, errSelector
			}
			value = strings.TrimSpace(value)
			if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
				value = value[1 : len(value)-1]
			}
			st.attrs = append(st.attrs, [2]string{key, value})
			st.anyValue = append(st.anyValue, !hasValue)
		default:
			return st, errSelector
		}
	}
	return st, nil
}

func attrValue(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Namespace == "" && strings.EqualFold(a.Key, key) {
			return a.Val, true
		}
	}
	return "", false
}

func (st step) matches(n *html.Node) bool {
	if n.Type != html.ElementNode || st.tag != "" && n.Data != st.tag {
		return false
	}
	if st.id != "" {
		if v, _ := attrValue(n, "id"); v != st.id {
			return false
		}
	}
	if len(st.classes) > 0 {
		v, _ := attrValue(n, "class")
		have := strings.Fields(v)
		for _, c := range st.classes {
			found := false
			for _, h := range have {
				found = found || h == c
			}
			if !found {
				return false
			}
		}
	}
	for i, a := range st.attrs {
		v, ok := attrValue(n, a[0])
		if !ok || !st.anyValue[i] && v != a[1] {
			return false
		}
	}
	return true
}

// matchSteps reports whether n matches the last step and its ancestors
// satisfy the earlier ones.
func matchSteps(steps []step, n *html.Node) bool {
	last := steps[len(steps)-1]
	if !last.matches(n) {
		return false
	}
	if len(steps) == 1 {
		return true
	}
	rest := steps[:len(steps)-1]
	for p := n.Parent; p != nil; p = p.Parent {
		if matchSteps(rest, p) {
			return true
		}
		if last.child {
			return false
		}
	}
	return false
}

func (sel selector) matches(n *html.Node) bool {
	for _, g := range sel {
		if matchSteps(g, n) {
			return true
		}
	}
	return false
}

// selectNodes returns matching elements in document order, outermost only
// (a match inside another match is already part of it).
func selectNodes(ctx context.Context, doc *html.Node, sel selector) ([]*html.Node, error) {
	var out []*html.Node
	visited := 0
	var walk func(*html.Node) error
	walk = func(n *html.Node) error {
		if visited++; visited%4096 == 0 && ctx.Err() != nil {
			return errors.New("webfetch cancelled or timed out")
		}
		if sel.matches(n) {
			out = append(out, n)
			return nil
		}
		for c := n.FirstChild; c != nil && len(out) < maxMatches; c = c.NextSibling {
			if err := walk(c); err != nil {
				return err
			}
		}
		return nil
	}
	return out, walk(doc)
}

func parseDocument(body []byte) (*html.Node, error) {
	doc, err := html.ParseWithOptions(bytes.NewReader(body), html.ParseOptionEnableScripting(false))
	if err != nil {
		return nil, errors.New("webfetch could not parse the HTML page")
	}
	return doc, nil
}

// renderSelected converts each match the way a whole page is converted,
// separated by a rule, and reports how many matched.
func renderSelected(ctx context.Context, nodes []*html.Node, format string, base *url.URL) (string, bool, error) {
	var parts []string
	truncated := false
	size := 0
	for _, n := range nodes {
		var b bytes.Buffer
		if format == "html" {
			if err := html.Render(&b, n); err != nil {
				return "", false, err
			}
			parts = append(parts, b.String())
		} else {
			wrapper := &html.Node{Type: html.ElementNode, Data: "body"}
			clone := cloneNode(n)
			wrapper.AppendChild(clone)
			if err := html.Render(&b, wrapper); err != nil {
				return "", false, err
			}
			text, cut, err := renderFrom(ctx, b.Bytes(), format, base, -1)
			if err != nil {
				return "", false, err
			}
			truncated = truncated || cut
			parts = append(parts, text)
		}
		if size += len(parts[len(parts)-1]); size > maxConverted {
			truncated = true
			break
		}
	}
	return strings.Join(parts, "\n\n---\n\n"), truncated, nil
}

func cloneNode(n *html.Node) *html.Node {
	c := &html.Node{Type: n.Type, DataAtom: n.DataAtom, Data: n.Data, Namespace: n.Namespace, Attr: n.Attr}
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		c.AppendChild(cloneNode(k))
	}
	return c
}

type pageLink struct {
	Text string `json:"text,omitempty"`
	URL  string `json:"url"`
}

// collectLinks lists the http(s) links inside roots, absolute and once each.
func collectLinks(roots []*html.Node, base *url.URL) []pageLink {
	links := []pageLink{}
	seen := map[string]bool{}
	size := 0
	var walk func(*html.Node) bool
	walk = func(n *html.Node) bool {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style" || n.Data == "template") {
			return true
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			href, _ := attrValue(n, "href")
			if link := safeLink(base, strings.TrimSpace(href)); link != "" && !seen[link] {
				seen[link] = true
				text := clipText(strings.Join(strings.Fields(nodeText(n)), " "), 120)
				if size += len(link) + len(text) + 24; size > linkBudget || len(links) == maxLinks {
					return false
				}
				links = append(links, pageLink{Text: text, URL: link})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if !walk(c) {
				return false
			}
		}
		return true
	}
	for _, r := range roots {
		if !walk(r) {
			break
		}
	}
	return links
}

func nodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if b.Len() > 512 {
			return
		}
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func clipText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && s[cut]&0xC0 == 0x80 {
		cut--
	}
	return s[:cut] + "…"
}
