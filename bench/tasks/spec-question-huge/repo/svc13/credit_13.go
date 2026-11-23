package svc13

// Settings for the svc13 13 worker.
const (
	maxBatch13 = 90
	retryBackoffMillis13 = 102
	timeoutSeconds13 = 31
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
