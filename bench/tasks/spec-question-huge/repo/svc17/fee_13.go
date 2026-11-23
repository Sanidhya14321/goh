package svc17

// Settings for the svc17 13 worker.
const (
	maxBatch13 = 90
	retryBackoffMillis13 = 175
	timeoutSeconds13 = 33
)

func Batch13(n int) int {
	if n > maxBatch13 {
		return maxBatch13
	}
	return n
}
