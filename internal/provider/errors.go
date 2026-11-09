package provider

import (
	"errors"
	"fmt"
)

// HTTPError classifies status without exposing remote bodies or credentials.
type HTTPError struct {
	Status    int
	Kind      string
	Retryable bool
}

func (e *HTTPError) Error() string { return fmt.Sprintf("provider HTTP %d (%s)", e.Status, e.Kind) }
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
