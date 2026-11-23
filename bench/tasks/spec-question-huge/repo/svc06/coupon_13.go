package svc06

// Settings for the svc06 13 worker.
const (
	maxBatch13 = 123
	retryBackoffMillis13 = 304
	timeoutSeconds13 = 57
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
