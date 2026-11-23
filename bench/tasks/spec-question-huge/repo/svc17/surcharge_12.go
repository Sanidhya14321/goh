package svc17

// Settings for the svc17 12 worker.
const (
	maxBatch12 = 471
	retryBackoffMillis12 = 394
	timeoutSeconds12 = 48
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
