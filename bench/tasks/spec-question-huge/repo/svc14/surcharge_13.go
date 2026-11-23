package svc14

// Settings for the svc14 13 worker.
const (
	maxBatch13 = 479
	retryBackoffMillis13 = 384
	timeoutSeconds13 = 16
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
