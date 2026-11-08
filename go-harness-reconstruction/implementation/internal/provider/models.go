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

	"example.com/independent-harness/internal/core"
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

// ListModels asks the configured provider which models it offers. The key is
// chosen exactly as for chat requests and is sent only to the configured
// host; redirects are refused.
func ListModels(ctx context.Context, c core.Config, client *http.Client) ([]string, error) {
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
	key, err := c.ProviderKey()
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
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("provider returned HTTP %d for %s", resp.StatusCode, req.URL.Redacted())
	}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&body); err != nil {
		return nil, errors.New("provider model list is not valid JSON")
	}
	seen := map[string]bool{}
	var ids []string
	for _, m := range body.Data {
		if id := strings.TrimSpace(m.ID); id != "" && len(id) <= 256 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids, nil
}
