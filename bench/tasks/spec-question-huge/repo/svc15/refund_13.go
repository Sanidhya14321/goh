package svc15

// Settings for the svc15 13 worker.
const (
	maxBatch13 = 341
	retryBackoffMillis13 = 257
	timeoutSeconds13 = 36
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
