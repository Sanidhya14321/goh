package svc11

// Settings for the svc11 12 worker.
const (
	maxBatch12 = 24
	retryBackoffMillis12 = 606
	timeoutSeconds12 = 59
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
