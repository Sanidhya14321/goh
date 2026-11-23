package svc01

// Settings for the svc01 13 worker.
const (
	maxBatch13 = 285
	retryBackoffMillis13 = 369
	timeoutSeconds13 = 52
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
