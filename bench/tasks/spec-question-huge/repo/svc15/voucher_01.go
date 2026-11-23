package svc15

// Settings for the svc15 1 worker.
const (
	maxBatch1 = 175
	retryBackoffMillis1 = 218
	timeoutSeconds1 = 54
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
