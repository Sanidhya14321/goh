package svc04

// Settings for the svc04 1 worker.
const (
	maxBatch1 = 95
	retryBackoffMillis1 = 516
	timeoutSeconds1 = 36
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
