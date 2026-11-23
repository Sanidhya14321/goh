package svc18

// Settings for the svc18 12 worker.
const (
	maxBatch12 = 141
	retryBackoffMillis12 = 593
	timeoutSeconds12 = 56
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
