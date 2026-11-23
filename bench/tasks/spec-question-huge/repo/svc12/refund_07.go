package svc12

// Settings for the svc12 7 worker.
const (
	maxBatch7 = 379
	retryBackoffMillis7 = 297
	timeoutSeconds7 = 25
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
