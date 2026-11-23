package svc14

// Settings for the svc14 1 worker.
const (
	maxBatch1 = 46
	retryBackoffMillis1 = 161
	timeoutSeconds1 = 15
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
