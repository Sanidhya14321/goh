package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
)

// ModelsURL derives the provider's model-list URL from its configured chat
// endpoint, so it always targets the same scheme and host.
func ModelsURL(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return "", errors.New("invalid provider endpoint")
	}
	for _, suffix := range []string{"/chat/completions", "/responses", "/messages"} {
		if strings.HasSuffix(u.Path, suffix) {
			u.Path = strings.TrimSuffix(u.Path, suffix) + "/models"
			u.RawQuery = ""
			return u.String(), nil
		}
	}
	return "", errors.New("cannot derive a models URL from this endpoint; set base_url to the API version path")
}

// ModelInfo is one entry of a provider's model list. Window is the context
// window when the provider reports one (OpenRouter's context_length, Groq's
// context_window, Anthropic's max_input_tokens), else 0.
type ModelInfo struct {
	ID     string
	Window int
}

// ErrKeyRejected reports that the provider refused the key (HTTP 401/403).
var ErrKeyRejected = errors.New("the provider rejected this API key")

// ListModels asks the configured provider which models it offers. The key is
// chosen exactly as for chat requests and is sent only to the configured
// host; redirects are refused.
func ListModels(ctx context.Context, c core.Config, client *http.Client) ([]string, error) {
	key, err := c.ProviderKey()
	if err != nil {
		return nil, err
	}
	infos, err := ListModelInfo(ctx, c, key, client)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(infos))
	for i, m := range infos {
		ids[i] = m.ID
	}
	return ids, nil
}

// ListModelInfo lists the provider's models using key, which need not be
// saved yet: provider setup uses it to check a pasted key before saving it.
func ListModelInfo(ctx context.Context, c core.Config, key string, client *http.Client) ([]ModelInfo, error) {
	if c.EffectiveProtocol() == core.ChatGPTProtocol {
		return nil, errors.New("Sign in with ChatGPT models come from your account; use /models in chat")
	}
	endpoint, err := c.ProviderEndpoint()
	if err != nil {
		return nil, err
	}
	target, err := ModelsURL(endpoint)
	if err != nil {
		return nil, err
	}
	if client == nil {
		client = http.DefaultClient
	}
	limited := *client
	limited.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	if c.EffectiveProtocol() == "anthropic_messages" {
		req.Header.Set("anthropic-version", "2023-06-01")
		if key != "" {
			req.Header.Set("x-api-key", key)
		}
	} else if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := limited.Do(req)
	if err != nil {
		return nil, fmt.Errorf("model list request failed: %w", err)
	}
	defer resp.Body.Close()
	if err := keyStatus(resp, req); err != nil {
		return nil, err
	}
	var body struct {
		Data []struct {
			ID             string `json:"id"`
			ContextLength  int    `json:"context_length"`
			ContextWindow  int    `json:"context_window"`
			MaxInputTokens int    `json:"max_input_tokens"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(&body); err != nil {
		return nil, errors.New("provider model list is not valid JSON")
	}
	seen := map[string]bool{}
	var out []ModelInfo
	for _, m := range body.Data {
		id := strings.TrimSpace(m.ID)
		if id == "" || len(id) > 256 || seen[id] || strings.ContainsAny(id, "\x00\r\n") {
			continue
		}
		seen[id] = true
		window := max(m.ContextLength, m.ContextWindow, m.MaxInputTokens)
		if window < 0 || window > 100_000_000 {
			window = 0
		}
		out = append(out, ModelInfo{ID: id, Window: window})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// keyStatus turns a refusal into ErrKeyRejected. Besides 401 and 403 this
// covers Google's 400 "Please pass a valid API key".
func keyStatus(resp *http.Response, req *http.Request) error {
	if resp.StatusCode/100 == 2 {
		return nil
	}
	rejected := resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden
	if resp.StatusCode == http.StatusBadRequest {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		lower := strings.ToLower(string(b))
		rejected = strings.Contains(lower, "api key") || strings.Contains(lower, "api_key")
	}
	if rejected {
		return fmt.Errorf("%w (HTTP %d from %s)", ErrKeyRejected, resp.StatusCode, req.URL.Host)
	}
	return fmt.Errorf("provider returned HTTP %d for %s", resp.StatusCode, req.URL.Redacted())
}

// CheckKey asks target (on the provider's own host) whether key is valid,
// for providers whose model list is public. Redirects are refused.
func CheckKey(ctx context.Context, c core.Config, target, key string, client *http.Client) error {
	endpoint, err := c.ProviderEndpoint()
	if err != nil {
		return err
	}
	a, errA := url.Parse(endpoint)
	b, errB := url.Parse(target)
	if errA != nil || errB != nil || a.Scheme != b.Scheme || a.Host != b.Host {
		return errors.New("key check URL must be on the provider's own host")
	}
	if client == nil {
		client = http.DefaultClient
	}
	limited := *client
	limited.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := limited.Do(req)
	if err != nil {
		return fmt.Errorf("key check failed: %w", err)
	}
	defer resp.Body.Close()
	return keyStatus(resp, req)
}
