package svc15

// Settings for the svc15 7 worker.
const (
	maxBatch7 = 291
	retryBackoffMillis7 = 234
	timeoutSeconds7 = 23
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
