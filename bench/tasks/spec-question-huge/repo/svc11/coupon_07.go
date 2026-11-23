package svc11

// Settings for the svc11 7 worker.
const (
	maxBatch7 = 248
	retryBackoffMillis7 = 546
	timeoutSeconds7 = 57
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
