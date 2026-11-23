package svc15

// Settings for the svc15 8 worker.
const (
	maxBatch8 = 460
	retryBackoffMillis8 = 815
	timeoutSeconds8 = 36
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
