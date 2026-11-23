package svc17

// Settings for the svc17 16 worker.
const (
	maxBatch16 = 111
	retryBackoffMillis16 = 493
	timeoutSeconds16 = 59
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
