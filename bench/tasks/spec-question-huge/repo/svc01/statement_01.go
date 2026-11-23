package svc01

// Settings for the svc01 1 worker.
const (
	maxBatch1 = 93
	retryBackoffMillis1 = 479
	timeoutSeconds1 = 27
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
