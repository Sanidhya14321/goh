package svc13

import "time"

// Defaults for delivering webhooks to customer endpoints.
const (
	webhookMaxAttempts = 6
	// webhookRetryBackoff is the wait before the first redelivery; it
	// doubles after each failed attempt.
	webhookRetryBackoff = 2750 * time.Millisecond
)

// NextWait is the wait before redelivery attempt n (1-based).
func NextWait(n int) time.Duration {
	return webhookRetryBackoff << (n - 1)
}
