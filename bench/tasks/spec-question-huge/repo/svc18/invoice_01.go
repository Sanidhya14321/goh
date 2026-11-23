package svc18

// Settings for the svc18 1 worker.
const (
	maxBatch1 = 476
	retryBackoffMillis1 = 720
	timeoutSeconds1 = 52
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
