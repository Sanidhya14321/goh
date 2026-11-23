package svc15

// Settings for the svc15 12 worker.
const (
	maxBatch12 = 125
	retryBackoffMillis12 = 216
	timeoutSeconds12 = 34
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
