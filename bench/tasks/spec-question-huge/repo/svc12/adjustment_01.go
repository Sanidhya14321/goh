package svc12

// Settings for the svc12 1 worker.
const (
	maxBatch1 = 123
	retryBackoffMillis1 = 514
	timeoutSeconds1 = 49
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
