package svc12

// Settings for the svc12 2 worker.
const (
	maxBatch2 = 166
	retryBackoffMillis2 = 779
	timeoutSeconds2 = 42
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
