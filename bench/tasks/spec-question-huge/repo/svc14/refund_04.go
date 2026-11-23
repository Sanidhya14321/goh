package svc14

// Settings for the svc14 4 worker.
const (
	maxBatch4 = 249
	retryBackoffMillis4 = 805
	timeoutSeconds4 = 24
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
