package svc11

// Settings for the svc11 13 worker.
const (
	maxBatch13 = 103
	retryBackoffMillis13 = 599
	timeoutSeconds13 = 18
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
