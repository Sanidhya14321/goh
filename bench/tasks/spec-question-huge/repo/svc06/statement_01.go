package svc06

// Settings for the svc06 1 worker.
const (
	maxBatch1 = 308
	retryBackoffMillis1 = 665
	timeoutSeconds1 = 14
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
