package webfetch

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

const maxHTMLTokens = 100000
const maxHTMLDepth = 256
const maxConverted = 64 << 10

// renderer writes converted text. Line breaks and inter-word spaces are owed
// rather than written, and paid only before the next visible content: nested
// blocks then cannot stack blank lines, lines carry no trailing spaces, and
// an element that turns out empty can be taken back whole (see mark/drop).
type renderer struct {
	buf       []byte
	truncated bool
	newlines  int  // line breaks owed before the next content (at most 2)
	space     bool // a space owed before the next inline content
	opened    bool // just after an opening marker: a leading space is dropped
}

type position struct {
	length, newlines int
	space, opened    bool
}

func (r *renderer) raw(s string) {
	if r.truncated {
		return
	}
	if available := maxConverted - len(r.buf); len(s) > available {
		s = s[:available]
		for !utf8.ValidString(s) {
			s = s[:len(s)-1]
		}
		r.truncated = true
	}
	r.buf = append(r.buf, s...)
}

// trailingNewlines counts the line breaks the output already ends with.
func (r *renderer) trailingNewlines() int {
	n := 0
	for i := len(r.buf) - 1; i >= 0 && r.buf[i] == '\n' && n < 2; i-- {
		n++
	}
	return n
}

// emit writes visible content after paying what is owed.
func (r *renderer) emit(s string) {
	if s == "" {
		return
	}
	if len(r.buf) > 0 {
		if r.newlines > 0 {
			r.raw(strings.Repeat("\n", max(0, r.newlines-r.trailingNewlines())))
		} else if r.space && !r.opened {
			r.raw(" ")
		}
	}
	r.newlines, r.space, r.opened = 0, false, false
	r.raw(s)
}
func (r *renderer) breakLine() {
	r.newlines, r.space = max(r.newlines, 1), false
}
func (r *renderer) paragraph() {
	r.newlines, r.space = 2, false
}

// open writes an opening inline marker ("[", "**"); a space right after it
// moves before it.
func (r *renderer) open(s string) {
	r.emit(s)
	r.opened = true
}

// close writes a closing marker; a space owed inside it moves after it.
func (r *renderer) close(s string) {
	space := r.space && !r.opened
	r.space, r.opened = false, false
	r.raw(s)
	r.space = space
}
func (r *renderer) mark() position {
	return position{len(r.buf), r.newlines, r.space, r.opened}
}

// visibleSince reports whether anything but whitespace and the element's own
// opening marker was written after p.
func (r *renderer) visibleSince(p position, opening string) bool {
	written := strings.TrimSpace(string(r.buf[p.length:]))
	return strings.TrimSpace(strings.TrimPrefix(written, strings.TrimSpace(opening))) != ""
}

// drop takes back everything written since p (an empty link or list item).
func (r *renderer) drop(p position) {
	if r.truncated {
		return
	}
	r.buf = r.buf[:p.length]
	r.newlines, r.space, r.opened = p.newlines, p.space, p.opened
}
func (r *renderer) text(s string, pre bool) {
	s = strings.ToValidUTF8(s, "\uFFFD")
	if pre {
		if s != "" {
			_, size := utf8.DecodeRuneInString(s)
			r.emit(s[:size])
			r.raw(s[size:])
		}
		return
	}
	start := -1
	for i, c := range s {
		if unicode.IsSpace(c) {
			if start >= 0 {
				r.emit(s[start:i])
				start = -1
			}
			if len(r.buf) > 0 && r.newlines == 0 {
				r.space = true
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		r.emit(s[start:])
	}
}

// Elements whose content is never page text.
func suppressedElement(name string) bool {
	switch name {
	case "script", "style", "noscript", "iframe", "object", "embed", "template", "svg", "canvas", "button", "select", "textarea", "dialog", "head", "title":
		return true
	}
	return false
}

// contentElement opens the page's own content, inside which header, footer
// and aside belong to it.
func contentElement(token html.Token) bool {
	return token.Data == "main" || token.Data == "article" || strings.EqualFold(attr(token, "role"), "main")
}

// chrome reports page furniture: site navigation, banners, footers, search
// boxes and content the page itself hides. header, footer and aside count as
// chrome only outside main content, where they frame the page rather than a
// document ("<main><header><h1>", "<article><aside>Note").
func chrome(token html.Token, inContent bool) bool {
	switch token.Data {
	case "nav":
		return true
	case "header", "footer", "aside":
		if !inContent {
			return true
		}
	}
	for _, a := range token.Attr {
		value := strings.ToLower(strings.TrimSpace(a.Val))
		switch a.Key {
		case "hidden":
			return true
		case "aria-hidden":
			if value == "true" {
				return true
			}
		case "role":
			switch value {
			case "navigation", "banner", "contentinfo", "search", "menu", "menubar", "dialog", "alertdialog":
				return true
			}
		case "style":
			if v := strings.ReplaceAll(value, " ", ""); strings.Contains(v, "display:none") || strings.Contains(v, "visibility:hidden") {
				return true
			}
		}
	}
	return false
}

func attr(token html.Token, key string) string {
	for _, a := range token.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasClass(token html.Token, class string) bool {
	for _, c := range strings.Fields(attr(token, "class")) {
		if c == class {
			return true
		}
	}
	return false
}

// outline is what a first pass learns about the page: its title and the one
// element that holds its main content, when the markup says so unambiguously.
type outline struct {
	title string
	root  int // token index of the content root's start tag; -1 for the whole page
}

// findOutline prefers, in order, a single rendered README/article body
// (article.markdown-body, as code hosts render repository READMEs), a single
// <main> (or role=main), then a single <article>. Several candidates of one
// kind are ambiguous and fall through to the next.
func findOutline(ctx context.Context, body []byte) (outline, error) {
	tokenizer := html.NewTokenizer(bytes.NewReader(body))
	tokenizer.SetMaxBuf(MaxResponse + 1)
	var readme, main, article []int
	type frame struct {
		name            string
		hidden, content bool
	}
	var stack []frame
	hidden, content := 0, 0
	o := outline{root: -1}
	inTitle := false
	for tokens := 0; ; tokens++ {
		if tokens > maxHTMLTokens {
			return o, errors.New("webfetch HTML token limit exceeded")
		}
		if ctx.Err() != nil {
			return o, errors.New("webfetch cancelled during HTML conversion")
		}
		kind := tokenizer.Next()
		if kind == html.ErrorToken {
			if tokenizer.Err() != io.EOF {
				return o, errors.New("webfetch HTML tokenizer failed")
			}
			break
		}
		token := tokenizer.Token()
		switch kind {
		case html.TextToken:
			if inTitle && len(o.title) < 512 {
				o.title += token.Data
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			if token.Data == "title" && o.title == "" && kind == html.StartTagToken {
				inTitle = true
			}
			// The same visibility the rendering pass applies: a candidate
			// inside hidden markup or page chrome is not the content.
			f := frame{name: token.Data}
			f.hidden = hidden > 0 || suppressedElement(token.Data) || chrome(token, content > 0)
			if !f.hidden {
				switch {
				case token.Data == "article" && hasClass(token, "markdown-body"):
					readme = append(readme, tokens)
					article = append(article, tokens)
				case token.Data == "article":
					article = append(article, tokens)
				case token.Data == "main" || strings.EqualFold(attr(token, "role"), "main"):
					main = append(main, tokens)
				}
			}
			if voidElement(token.Data) || kind == html.SelfClosingTagToken && (token.Data == "svg" || token.Data == "math") {
				continue
			}
			f.content = contentElement(token)
			if len(stack) >= maxHTMLDepth {
				return o, errors.New("webfetch HTML nesting exceeds 256 levels")
			}
			if f.hidden {
				hidden++
			}
			if f.content {
				content++
			}
			stack = append(stack, f)
		case html.EndTagToken:
			if token.Data == "title" {
				inTitle = false
			}
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].name == token.Data {
					for _, f := range stack[i:] {
						if f.hidden {
							hidden--
						}
						if f.content {
							content--
						}
					}
					stack = stack[:i]
					break
				}
			}
		}
	}
	o.title = strings.Join(strings.Fields(strings.ToValidUTF8(o.title, "\uFFFD")), " ")
	for _, candidates := range [][]int{readme, main, article} {
		if len(candidates) == 1 {
			o.root = candidates[0]
			break
		}
	}
	return o, nil
}

// renderHTML converts a page to text or Markdown: its main content when the
// markup identifies it (see findOutline), without navigation, page chrome,
// empty links or list items, and with at most one blank line in a row
// outside preformatted text. The page title leads when known.
func renderHTML(ctx context.Context, body []byte, format string, base *url.URL) (string, bool, error) {
	o, err := findOutline(ctx, body)
	if err != nil {
		return "", false, err
	}
	content, truncated, err := renderFrom(ctx, body, format, base, o.root)
	if err == nil && o.root >= 0 && content == "" {
		// The marked content was empty (a client-rendered shell): fall back
		// to the whole page.
		content, truncated, err = renderFrom(ctx, body, format, base, -1)
	}
	if err != nil {
		return "", false, err
	}
	if o.title != "" && !strings.HasPrefix(content, o.title) && !strings.HasPrefix(content, "# "+o.title) {
		title := "Title: " + o.title + "\n\n"
		if len(title)+len(content) > maxConverted {
			content = strings.ToValidUTF8(content[:max(0, maxConverted-len(title))], "")
			truncated = true
		}
		content = strings.TrimSpace(title + content)
	}
	return content, truncated, nil
}

func renderFrom(ctx context.Context, body []byte, format string, base *url.URL, root int) (string, bool, error) {
	tokenizer := html.NewTokenizer(bytes.NewReader(body))
	tokenizer.SetMaxBuf(MaxResponse + 1)
	type frame struct {
		name, link    string
		hidden, pre   bool
		content, root bool
		collapsible   bool
		start         position
		opening       string
	}
	var stack []frame
	r := &renderer{}
	markdown := format == "markdown"
	// outside counts as one more hidden level until the content root opens.
	hidden, pre, content, outside := 0, 0, 0, 0
	if root >= 0 {
		outside = 1
	}
	rootClosed := false
	closeFrame := func(f frame) {
		if f.content {
			content--
		}
		if f.root {
			outside, rootClosed = 1, true
		}
		if f.hidden {
			hidden--
			return
		}
		if hidden > 0 || outside > 0 {
			return
		}
		if f.pre {
			pre--
		}
		if f.collapsible && !r.visibleSince(f.start, f.opening) {
			r.drop(f.start)
			if f.name == "li" || f.name == "tr" || strings.HasPrefix(f.opening, "#") {
				r.breakLine()
			}
			return
		}
		if markdown {
			switch f.name {
			case "a":
				if f.link != "" && pre == 0 {
					r.close("](" + f.link + ")")
				}
			case "strong", "b":
				if pre == 0 {
					r.close("**")
				}
			case "em", "i":
				if pre == 0 {
					r.close("*")
				}
			case "code":
				if pre == 0 {
					r.close("`")
				}
			case "pre":
				r.breakLine()
				r.emit("```")
				r.paragraph()
			case "p", "div", "section", "article", "main", "blockquote", "ul", "ol", "table", "details", "figure", "h1", "h2", "h3", "h4", "h5", "h6":
				r.paragraph()
			case "li", "tr", "summary", "dt", "dd":
				r.breakLine()
			case "td", "th":
				r.space = true
			}
		} else {
			switch f.name {
			case "p", "div", "li", "section", "article", "main", "table", "details", "summary", "dt", "dd", "h1", "h2", "h3", "h4", "h5", "h6", "tr", "pre", "ul", "ol", "blockquote", "figure":
				r.breakLine()
			case "td", "th":
				r.space = true
			}
		}
	}
	for tokens := 0; ; tokens++ {
		if tokens > maxHTMLTokens {
			return "", false, errors.New("webfetch HTML token limit exceeded")
		}
		if ctx.Err() != nil {
			return "", false, errors.New("webfetch cancelled during HTML conversion")
		}
		kind := tokenizer.Next()
		if kind == html.ErrorToken {
			if tokenizer.Err() != io.EOF {
				return "", false, errors.New("webfetch HTML tokenizer failed")
			}
			break
		}
		token := tokenizer.Token()
		switch kind {
		case html.TextToken:
			if hidden == 0 && outside == 0 {
				r.text(token.Data, pre > 0)
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			name := token.Data
			f := frame{name: name}
			if tokens == root {
				f.root, outside = true, 0
			}
			f.hidden = hidden > 0 || suppressedElement(name) || (!f.root && chrome(token, content > 0 || outside == 0 && root >= 0))
			if contentElement(token) {
				f.content = true
				content++
			}
			if f.hidden {
				hidden++
			} else if outside == 0 {
				if name == "pre" {
					f.pre = true
					pre++
				}
				f.start = r.mark()
				switch name {
				case "a", "li", "strong", "b", "em", "i", "code", "h1", "h2", "h3", "h4", "h5", "h6", "tr", "summary", "dt", "dd":
					f.collapsible = pre == 0
				}
				if markdown {
					switch name {
					case "h1", "h2", "h3", "h4", "h5", "h6":
						r.paragraph()
						f.start = r.mark()
						f.opening = strings.Repeat("#", int(name[1]-'0'))
						r.open(f.opening + " ")
					case "p", "div", "section", "article", "main", "ul", "ol", "blockquote", "table", "details", "figure":
						r.paragraph()
					case "br":
						r.breakLine()
					case "hr":
						r.paragraph()
						r.emit("---")
						r.paragraph()
					case "li":
						r.breakLine()
						f.start = r.mark()
						f.opening = "-"
						r.open("- ")
					case "tr", "summary", "dt", "dd":
						r.breakLine()
						f.start = r.mark()
					case "strong", "b":
						if pre == 0 {
							f.opening = "**"
							r.open("**")
						}
					case "em", "i":
						if pre == 0 {
							f.opening = "*"
							r.open("*")
						}
					case "pre":
						r.paragraph()
						r.emit("```\n")
						r.opened = false
					case "code":
						if pre == 0 {
							f.opening = "`"
							r.open("`")
						}
					case "a":
						f.link = safeLink(base, attr(token, "href"))
						if f.link != "" && pre == 0 {
							f.opening = "["
							r.open("[")
						}
					case "img":
						r.text(attr(token, "alt"), false)
					}
				} else {
					switch name {
					case "br", "hr", "p", "div", "li", "tr", "h1", "h2", "h3", "h4", "h5", "h6", "pre", "ul", "ol", "table", "blockquote", "summary", "dt", "dd":
						r.breakLine()
						f.start = r.mark()
					}
				}
			}
			// "<svg/>" is complete; "<script/>" still opens a script (HTML
			// ignores the slash on its own elements).
			if voidElement(name) || kind == html.SelfClosingTagToken && (name == "svg" || name == "math") {
				closeFrame(f)
			} else {
				if len(stack) >= maxHTMLDepth {
					return "", false, errors.New("webfetch HTML nesting exceeds 256 levels")
				}
				stack = append(stack, f)
			}
		case html.EndTagToken:
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].name == token.Data {
					for len(stack) > i {
						f := stack[len(stack)-1]
						stack = stack[:len(stack)-1]
						closeFrame(f)
					}
					break
				}
			}
		}
		if r.truncated || rootClosed {
			break
		}
	}
	return strings.TrimSpace(string(r.buf)), r.truncated, nil
}
func voidElement(name string) bool {
	switch name {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	}
	return false
}
func safeLink(base *url.URL, raw string) string {
	if len(raw) > 2048 {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	u = base.ResolveReference(u)
	if (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return ""
	}
	link := u.String()
	if strings.ContainsAny(link, "()<>\n\r\t") {
		return ""
	}
	return link
}
