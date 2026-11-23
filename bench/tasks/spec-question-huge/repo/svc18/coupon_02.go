package svc18

// Settings for the svc18 2 worker.
const (
	maxBatch2 = 436
	retryBackoffMillis2 = 393
	timeoutSeconds2 = 54
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
