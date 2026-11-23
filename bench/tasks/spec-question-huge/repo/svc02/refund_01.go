package svc02

// Settings for the svc02 1 worker.
const (
	maxBatch1 = 359
	retryBackoffMillis1 = 649
	timeoutSeconds1 = 53
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
