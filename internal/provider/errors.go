package provider

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// HTTPError classifies status without exposing remote bodies or credentials.
type HTTPError struct {
	Status    int
	Kind      string
	Retryable bool
	// Limit and Requested are the per-request token cap and the size of
	// the refused request, when a 413 states them (Groq: "Limit 8000,
	// Requested 9512"). Zero when unknown.
	Limit     int
	Requested int
	// Wait is how long a rate limit says to wait when that is longer than
	// goh retries for (a daily token quota); zero otherwise.
	Wait time.Duration
}

func (e *HTTPError) Error() string {
	message := fmt.Sprintf("provider HTTP %d (%s)", e.Status, e.Kind)
	if e.Wait > 0 {
		return message + fmt.Sprintf(": the provider's limit resets in about %s (often a daily or monthly token quota); try again then, or use another model or plan", strings.TrimSuffix(e.Wait.Round(time.Minute).String(), "0s"))
	}
	if e.TooLarge() && e.Limit > 0 {
		message += fmt.Sprintf(": the provider accepts at most %d tokens per request", e.Limit)
		if e.Requested > 0 {
			message += fmt.Sprintf(" and this one needed %d", e.Requested)
		}
		return message + " (often a free-tier token-per-minute cap); use a model or plan with a higher limit"
	}
	if e.Status == 413 {
		// Usually a per-request token cap below the configured context
		// window (Groq's free tier allows about 8k tokens per request,
		// output reserve included).
		message += ": the request is larger than this provider accepts at once (often a free-tier token-per-minute cap); use a model or plan with a higher limit, or a context_window the provider really accepts"
	}
	return message
}

// TooLarge reports a request the provider will not accept at its current
// size: HTTP 413, or a rate limit whose stated cap is below the request.
func (e *HTTPError) TooLarge() bool {
	return e.Status == 413 || e.Status == 429 && e.Limit > 0 && e.Requested > e.Limit
}
func classifyHTTP(status int) *HTTPError {
	kind := "request"
	retry := false
	switch {
	case status == 401 || status == 403:
		kind = "authentication"
	case status == 408:
		kind = "timeout"
		retry = true
	case status == 413:
		kind = "request_too_large"
	case status == 429:
		kind = "rate_limit"
		retry = true
	case status >= 500:
		kind = "server"
		retry = true
	case status >= 300 && status < 400:
		kind = "redirect_refused"
	}
	return &HTTPError{Status: status, Kind: kind, Retryable: retry}
}

// Failoverable reports failures that happened before the provider returned any
// response body: connection failures and retryable HTTP statuses after the
// client's own bounded retries. No content, tool call or partial stream exists
// for these, so another explicitly configured route may safely receive the turn.
func Failoverable(err error) bool {
	if errors.Is(err, ErrConnection) {
		return true
	}
	var h *HTTPError
	return errors.As(err, &h) && h.Retryable
}

var (
	capLimit     = regexp.MustCompile(`(?i)\blimit[\s:=]+(\d{3,9})\b`)
	capRequested = regexp.MustCompile(`(?i)\brequested[\s:=]+(\d{3,9})\b`)
)

var capWait = regexp.MustCompile(`(?i)\btry again in ([0-9hms.]{2,24})`)

// requestCap reads the token cap, request size and wait a 413 or 429 body
// states. Bodies can reflect request data, so only these values are taken
// from it.
func requestCap(body io.Reader) (limit, requested int, wait time.Duration) {
	b, _ := io.ReadAll(io.LimitReader(body, 8<<10))
	if m := capWait.FindSubmatch(b); m != nil {
		if d, err := time.ParseDuration(strings.TrimRight(string(m[1]), ".")); err == nil && d > 0 {
			wait = d
		}
	}
	number := func(re *regexp.Regexp) int {
		if m := re.FindSubmatch(b); m != nil {
			n, _ := strconv.Atoi(string(m[1]))
			return n
		}
		return 0
	}
	return number(capLimit), number(capRequested), wait
}
