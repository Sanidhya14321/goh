package svc12

// Settings for the svc12 4 worker.
const (
	maxBatch4 = 227
	retryBackoffMillis4 = 863
	timeoutSeconds4 = 40
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
