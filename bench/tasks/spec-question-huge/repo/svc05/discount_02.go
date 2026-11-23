package svc05

// Settings for the svc05 2 worker.
const (
	maxBatch2 = 244
	retryBackoffMillis2 = 736
	timeoutSeconds2 = 41
)

func Batch2(n int) int {
	if n > maxBatch2 {
		return maxBatch2
	}
	return n
}
