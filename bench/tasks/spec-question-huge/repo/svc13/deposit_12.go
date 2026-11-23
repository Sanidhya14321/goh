package svc13

// Settings for the svc13 12 worker.
const (
	maxBatch12 = 132
	retryBackoffMillis12 = 277
	timeoutSeconds12 = 40
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
