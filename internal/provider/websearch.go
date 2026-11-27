package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
)

// WebSource is one page a provider search cited.
type WebSource struct {
	Title string `json:"title,omitempty"`
	URL   string `json:"url"`
}

// WebAnswer is a provider search's short answer and the pages it cited.
type WebAnswer struct {
	Answer  string      `json:"answer"`
	Sources []WebSource `json:"sources"`
}

const webSearchInstructions = "Search the web for the query and answer it briefly from what you find: " +
	"the facts, versions, names and numbers that answer it, citing the pages. Do not speculate beyond the sources."

const maxWebAnswer = 8 << 10

// SupportsWebSearch reports whether the configured provider has a hosted
// search tool goh can call: the OpenAI Responses API (an API key or the
// ChatGPT sign-in) or the Anthropic Messages API.
func SupportsWebSearch(c core.Config) bool {
	switch c.EffectiveProtocol() {
	case core.ChatGPTProtocol, "openai_responses", "anthropic_messages":
		return true
	}
	return false
}

// WebSearch runs one query through the provider's hosted search tool, in a
// request of its own (the conversation and its prompt cache are untouched).
func (c *Client) WebSearch(ctx context.Context, query string) (WebAnswer, error) {
	if !SupportsWebSearch(c.Config) {
		return WebAnswer{}, errors.New("this provider has no hosted web search; set websearch_url to a SearXNG endpoint instead")
	}
	endpoint, err := c.Config.ProviderEndpoint()
	if err != nil {
		return WebAnswer{}, err
	}
	chatGPT := c.Config.EffectiveProtocol() == core.ChatGPTProtocol
	anthropic := c.Config.EffectiveProtocol() == "anthropic_messages"
	key, err := c.Config.ProviderKey()
	if chatGPT {
		key, err = core.ChatGPTAccessToken(ctx)
	}
	if err != nil {
		return WebAnswer{}, err
	}
	var body map[string]any
	if anthropic {
		body = map[string]any{"model": c.Config.Model, "max_tokens": 2048, "system": webSearchInstructions,
			"messages": []any{map[string]any{"role": "user", "content": query}},
			"tools":    []any{map[string]any{"type": "web_search_20250305", "name": "web_search", "max_uses": 3}}}
	} else {
		body = map[string]any{"model": c.Config.Model, "stream": true, "store": false,
			"input": []any{map[string]any{"role": "developer", "content": webSearchInstructions},
				map[string]any{"role": "user", "content": query}},
			"tools": []any{map[string]any{"type": "web_search"}}}
		if c.Config.ReasoningEffort != "" {
			body["reasoning"] = map[string]any{"effort": "low"}
		}
	}
	b, err := json.Marshal(body)
	if err != nil {
		return WebAnswer{}, err
	}
	if c.HTTP == nil {
		return WebAnswer{}, errors.New("provider HTTP client unavailable")
	}
	client := *c.HTTP
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	for renewed := false; ; renewed = true {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
		if err != nil {
			return WebAnswer{}, errors.New("invalid provider request configuration")
		}
		req.Header.Set("Content-Type", "application/json")
		if anthropic {
			req.Header.Set("anthropic-version", "2023-06-01")
			if key != "" {
				req.Header.Set("x-api-key", key)
			}
		} else {
			req.Header.Set("Accept", "text/event-stream")
			if key != "" {
				req.Header.Set("Authorization", "Bearer "+key)
			}
		}
		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return WebAnswer{}, ctx.Err()
			}
			return WebAnswer{}, ErrConnection
		}
		if resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			if anthropic {
				return decodeAnthropicSearch(io.LimitReader(resp.Body, maxResponse))
			}
			return readResponsesSearch(resp.Body)
		}
		resp.Body.Close()
		if chatGPT && resp.StatusCode == http.StatusUnauthorized && !renewed {
			if next, e := core.RenewChatGPTAccessToken(ctx, key); e == nil && next != "" {
				key = next
				continue
			}
		}
		// Bodies can reflect credentials; only the status is reported.
		return WebAnswer{}, fmt.Errorf("provider web search failed: HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
}

// readResponsesSearch reads a Responses stream to its completed event.
func readResponsesSearch(r io.Reader) (WebAnswer, error) {
	scanner := bufio.NewScanner(io.LimitReader(r, maxResponse+1))
	scanner.Buffer(make([]byte, 4096), maxResponse)
	var data []string
	var items []json.RawMessage // the ChatGPT service streams items and completes with none
	consume := func() (WebAnswer, bool, error) {
		raw := strings.Join(data, "\n")
		data = nil
		var event struct {
			Type     string          `json:"type"`
			Item     json.RawMessage `json:"item"`
			Response struct {
				Output []json.RawMessage `json:"output"`
			} `json:"response"`
		}
		if raw == "" || raw == "[DONE]" || json.Unmarshal([]byte(raw), &event) != nil {
			return WebAnswer{}, false, nil
		}
		switch event.Type {
		case "response.output_item.done":
			if len(event.Item) > 0 && len(items) < 64 {
				items = append(items, event.Item)
			}
		case "response.completed":
			output := event.Response.Output
			if len(output) == 0 {
				output = items
			}
			a, err := decodeResponsesSearch(output)
			return a, true, err
		case "response.incomplete":
			return WebAnswer{}, true, errors.New("provider web search did not finish")
		case "response.failed", "error":
			return WebAnswer{}, true, responsesStreamError(raw)
		}
		return WebAnswer{}, false, nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if a, done, err := consume(); done {
				return a, err
			}
		} else if strings.HasPrefix(line, "data:") {
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if a, done, err := consume(); done {
		return a, err
	}
	return WebAnswer{}, ErrInterrupted
}

func decodeResponsesSearch(output []json.RawMessage) (WebAnswer, error) {
	var text strings.Builder
	var sources []WebSource
	for _, raw := range output {
		var item struct {
			Type    string `json:"type"`
			Content []struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				Annotations []struct {
					Type  string `json:"type"`
					URL   string `json:"url"`
					Title string `json:"title"`
				} `json:"annotations"`
			} `json:"content"`
		}
		if json.Unmarshal(raw, &item) != nil || item.Type != "message" {
			continue // reasoning and web_search_call items carry nothing to return
		}
		for _, part := range item.Content {
			if part.Type != "output_text" {
				continue
			}
			text.WriteString(part.Text)
			for _, a := range part.Annotations {
				if a.Type == "url_citation" {
					sources = append(sources, WebSource{Title: a.Title, URL: a.URL})
				}
			}
		}
	}
	return webAnswer(text.String(), sources)
}

func decodeAnthropicSearch(r io.Reader) (WebAnswer, error) {
	var msg struct {
		Content []struct {
			Type      string `json:"type"`
			Text      string `json:"text"`
			Citations []struct {
				Type  string `json:"type"`
				URL   string `json:"url"`
				Title string `json:"title"`
			} `json:"citations"`
			Content json.RawMessage `json:"content"`
		} `json:"content"`
	}
	if err := json.NewDecoder(r).Decode(&msg); err != nil {
		return WebAnswer{}, errors.New("provider web search returned an unreadable reply")
	}
	var text strings.Builder
	var sources []WebSource
	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			text.WriteString(block.Text)
			for _, c := range block.Citations {
				if c.Type == "web_search_result_location" {
					sources = append(sources, WebSource{Title: c.Title, URL: c.URL})
				}
			}
		case "web_search_tool_result":
			var results []struct {
				Type  string `json:"type"`
				URL   string `json:"url"`
				Title string `json:"title"`
			}
			if json.Unmarshal(block.Content, &results) == nil {
				for _, r := range results {
					if r.Type == "web_search_result" {
						sources = append(sources, WebSource{Title: r.Title, URL: r.URL})
					}
				}
			}
		}
	}
	return webAnswer(text.String(), sources)
}

// webAnswer bounds the answer and keeps each http(s) source once, in order.
func webAnswer(text string, sources []WebSource) (WebAnswer, error) {
	text = strings.TrimSpace(strings.ToValidUTF8(text, ""))
	if text == "" {
		return WebAnswer{}, errors.New("provider web search returned no answer")
	}
	if len(text) > maxWebAnswer {
		cut := maxWebAnswer
		for cut > 0 && text[cut]&0xC0 == 0x80 {
			cut--
		}
		text = text[:cut] + "…"
	}
	a := WebAnswer{Answer: text, Sources: []WebSource{}}
	seen := map[string]bool{}
	for _, s := range sources {
		u, err := url.Parse(s.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || seen[s.URL] || len(s.URL) > 2000 {
			continue
		}
		seen[s.URL] = true
		if len(s.Title) > 200 {
			s.Title = strings.ToValidUTF8(s.Title[:200], "") + "…"
		}
		a.Sources = append(a.Sources, s)
		if len(a.Sources) == 10 {
			break
		}
	}
	return a, nil
}

// SupportsWebSearch reports whether this client's provider has hosted search.
func (c *Client) SupportsWebSearch() bool { return SupportsWebSearch(c.Config) }

// WebSearchJSON is WebSearch encoded for the websearch tool result.
func (c *Client) WebSearchJSON(ctx context.Context, query string) (json.RawMessage, error) {
	a, err := c.WebSearch(ctx, query)
	if err != nil {
		return nil, err
	}
	return json.Marshal(a)
}
