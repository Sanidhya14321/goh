package svc12

// Settings for the svc12 13 worker.
const (
	maxBatch13 = 164
	retryBackoffMillis13 = 114
	timeoutSeconds13 = 50
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
