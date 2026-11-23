package svc17

// Settings for the svc17 2 worker.
const (
	maxBatch2 = 237
	retryBackoffMillis2 = 270
	timeoutSeconds2 = 49
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
