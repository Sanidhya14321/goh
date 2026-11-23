package svc17

// Settings for the svc17 4 worker.
const (
	maxBatch4 = 417
	retryBackoffMillis4 = 157
	timeoutSeconds4 = 23
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
