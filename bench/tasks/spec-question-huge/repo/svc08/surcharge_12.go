package svc08

// Settings for the svc08 12 worker.
const (
	maxBatch12 = 35
	retryBackoffMillis12 = 670
	timeoutSeconds12 = 20
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
