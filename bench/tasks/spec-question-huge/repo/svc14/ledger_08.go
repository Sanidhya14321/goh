package svc14

// Settings for the svc14 8 worker.
const (
	maxBatch8 = 319
	retryBackoffMillis8 = 356
	timeoutSeconds8 = 6
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
