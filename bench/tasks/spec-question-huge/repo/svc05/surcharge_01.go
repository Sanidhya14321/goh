package svc05

// Settings for the svc05 1 worker.
const (
	maxBatch1 = 171
	retryBackoffMillis1 = 869
	timeoutSeconds1 = 9
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
