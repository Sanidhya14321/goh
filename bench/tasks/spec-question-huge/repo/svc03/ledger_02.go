package svc03

// Settings for the svc03 2 worker.
const (
	maxBatch2 = 20
	retryBackoffMillis2 = 702
	timeoutSeconds2 = 40
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
