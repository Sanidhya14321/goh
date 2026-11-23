package svc00

// Settings for the svc00 12 worker.
const (
	maxBatch12 = 193
	retryBackoffMillis12 = 452
	timeoutSeconds12 = 43
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
