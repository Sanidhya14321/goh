package svc17

// Settings for the svc17 11 worker.
const (
	maxBatch11 = 140
	retryBackoffMillis11 = 782
	timeoutSeconds11 = 5
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
