package svc00

// Settings for the svc00 2 worker.
const (
	maxBatch2 = 356
	retryBackoffMillis2 = 858
	timeoutSeconds2 = 39
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
