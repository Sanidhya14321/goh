package svc15

// Settings for the svc15 4 worker.
const (
	maxBatch4 = 37
	retryBackoffMillis4 = 292
	timeoutSeconds4 = 38
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
