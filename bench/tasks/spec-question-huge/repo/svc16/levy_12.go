package svc16

// Settings for the svc16 12 worker.
const (
	maxBatch12 = 476
	retryBackoffMillis12 = 567
	timeoutSeconds12 = 5
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
