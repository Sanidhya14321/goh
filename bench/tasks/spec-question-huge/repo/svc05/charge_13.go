package svc05

// Settings for the svc05 13 worker.
const (
	maxBatch13 = 361
	retryBackoffMillis13 = 749
	timeoutSeconds13 = 59
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
