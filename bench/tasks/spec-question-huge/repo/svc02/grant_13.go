package svc02

// Settings for the svc02 13 worker.
const (
	maxBatch13 = 494
	retryBackoffMillis13 = 180
	timeoutSeconds13 = 10
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
