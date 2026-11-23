package svc09

// Settings for the svc09 13 worker.
const (
	maxBatch13 = 419
	retryBackoffMillis13 = 717
	timeoutSeconds13 = 19
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
