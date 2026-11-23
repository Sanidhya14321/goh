package svc07

// Settings for the svc07 13 worker.
const (
	maxBatch13 = 356
	retryBackoffMillis13 = 866
	timeoutSeconds13 = 16
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
