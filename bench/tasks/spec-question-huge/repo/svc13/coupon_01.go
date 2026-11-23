package svc13

// Settings for the svc13 1 worker.
const (
	maxBatch1 = 47
	retryBackoffMillis1 = 690
	timeoutSeconds1 = 45
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
