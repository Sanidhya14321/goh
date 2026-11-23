package svc12

// Settings for the svc12 8 worker.
const (
	maxBatch8 = 390
	retryBackoffMillis8 = 648
	timeoutSeconds8 = 53
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
