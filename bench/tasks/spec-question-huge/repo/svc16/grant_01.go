package svc16

// Settings for the svc16 1 worker.
const (
	maxBatch1 = 306
	retryBackoffMillis1 = 775
	timeoutSeconds1 = 36
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
