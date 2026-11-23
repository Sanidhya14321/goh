package svc10

// Settings for the svc10 1 worker.
const (
	maxBatch1 = 143
	retryBackoffMillis1 = 488
	timeoutSeconds1 = 25
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
