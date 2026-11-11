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

type renderer struct {
	b         strings.Builder
	truncated bool
	last      rune
}

func (r *renderer) write(s string) {
	if r.truncated {
		return
	}
	available := maxConverted - r.b.Len()
	if len(s) > available {
		s = s[:available]
		for !utf8.ValidString(s) {
			s = s[:len(s)-1]
		}
		r.truncated = true
	}
	r.b.WriteString(s)
	if s != "" {
		r.last, _ = utf8.DecodeLastRuneInString(s)
	}
}
func (r *renderer) breakLine() {
	if r.b.Len() > 0 && r.last != '\n' {
		r.write("\n")
	}
}
func (r *renderer) paragraph() {
	r.breakLine()
	if r.b.Len() > 0 {
		r.write("\n")
	}
}
func (r *renderer) text(s string, pre bool) {
	if pre {
		r.write(s)
		return
	}
	space := false
	for _, c := range s {
		if unicode.IsSpace(c) {
			space = true
			continue
		}
		if space && r.b.Len() > 0 && !unicode.IsSpace(r.last) {
			r.write(" ")
		}
		space = false
		r.write(string(c))
	}
	if space && r.b.Len() > 0 && !unicode.IsSpace(r.last) {
		r.write(" ")
	}
}
func renderHTML(ctx context.Context, body []byte, format string, base *url.URL) (string, bool, error) {
	tokenizer := html.NewTokenizer(bytes.NewReader(body))
	tokenizer.SetMaxBuf(MaxResponse + 1)
	type frame struct {
		name, link  string
		hidden, pre bool
	}
	var stack []frame
	r := &renderer{}
	hidden, pre := 0, 0
	closeFrame := func(f frame) {
		if f.hidden {
			hidden--
			return
		}
		if hidden > 0 {
			return
		}
		if f.pre {
			pre--
		}
		if format == "markdown" {
			switch f.name {
			case "a":
				if f.link != "" {
					r.write("](" + f.link + ")")
				}
			case "strong", "b":
				r.write("**")
			case "em", "i":
				r.write("*")
			case "code":
				if pre == 0 {
					r.write("`")
				}
			case "pre":
				r.breakLine()
				r.write("```\n")
			case "p", "div", "section", "article", "blockquote", "ul", "ol", "h1", "h2", "h3", "h4", "h5", "h6":
				r.paragraph()
			case "li", "tr":
				r.breakLine()
			}
		} else {
			switch f.name {
			case "p", "div", "li", "section", "article", "h1", "h2", "h3", "h4", "h5", "h6", "tr", "pre":
				r.breakLine()
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
			if hidden == 0 {
				r.text(token.Data, pre > 0)
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			name := token.Data
			f := frame{name: name}
			switch name {
			case "script", "style", "noscript", "iframe", "object", "embed", "template":
				f.hidden = true
			}
			if hidden > 0 {
				f.hidden = true
			}
			if f.hidden {
				hidden++
			} else {
				if name == "pre" {
					f.pre = true
					pre++
				}
				if format == "markdown" {
					switch name {
					case "h1", "h2", "h3", "h4", "h5", "h6":
						r.paragraph()
						r.write(strings.Repeat("#", int(name[1]-'0')) + " ")
					case "p", "div", "section", "article", "ul", "ol", "blockquote":
						r.paragraph()
					case "br":
						r.breakLine()
					case "hr":
						r.paragraph()
						r.write("---\n")
					case "li":
						r.breakLine()
						r.write("- ")
					case "strong", "b":
						r.write("**")
					case "em", "i":
						r.write("*")
					case "pre":
						r.paragraph()
						r.write("```\n")
					case "code":
						if pre == 0 {
							r.write("`")
						}
					case "a":
						for _, a := range token.Attr {
							if a.Key == "href" {
								f.link = safeLink(base, a.Val)
							}
						}
						if f.link != "" {
							r.write("[")
						}
					case "img":
						for _, a := range token.Attr {
							if a.Key == "alt" {
								r.text(a.Val, false)
							}
						}
					}
				} else {
					switch name {
					case "br", "hr", "p", "div", "li", "tr", "h1", "h2", "h3", "h4", "h5", "h6", "pre":
						r.breakLine()
					}
				}
			}
			void := voidElement(name)
			if void {
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
		if r.truncated {
			break
		}
	}
	return strings.TrimSpace(r.b.String()), r.truncated, nil
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
