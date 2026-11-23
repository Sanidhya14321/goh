package svc12

// Settings for the svc12 12 worker.
const (
	maxBatch12 = 161
	retryBackoffMillis12 = 332
	timeoutSeconds12 = 28
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
