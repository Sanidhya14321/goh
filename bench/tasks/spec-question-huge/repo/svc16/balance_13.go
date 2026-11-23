package svc16

// Settings for the svc16 13 worker.
const (
	maxBatch13 = 219
	retryBackoffMillis13 = 770
	timeoutSeconds13 = 14
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
