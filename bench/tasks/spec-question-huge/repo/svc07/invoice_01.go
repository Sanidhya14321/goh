package svc07

// Settings for the svc07 1 worker.
const (
	maxBatch1 = 65
	retryBackoffMillis1 = 710
	timeoutSeconds1 = 32
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
