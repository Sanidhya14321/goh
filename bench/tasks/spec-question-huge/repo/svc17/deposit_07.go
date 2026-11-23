package svc17

// Settings for the svc17 7 worker.
const (
	maxBatch7 = 97
	retryBackoffMillis7 = 279
	timeoutSeconds7 = 10
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
