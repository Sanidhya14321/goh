package svc07

// Settings for the svc07 8 worker.
const (
	maxBatch8 = 329
	retryBackoffMillis8 = 421
	timeoutSeconds8 = 47
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
