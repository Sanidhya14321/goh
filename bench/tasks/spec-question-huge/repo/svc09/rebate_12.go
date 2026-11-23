package svc09

// Settings for the svc09 12 worker.
const (
	maxBatch12 = 285
	retryBackoffMillis12 = 865
	timeoutSeconds12 = 52
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
