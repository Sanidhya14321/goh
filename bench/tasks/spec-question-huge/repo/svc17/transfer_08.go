package svc17

// Settings for the svc17 8 worker.
const (
	maxBatch8 = 455
	retryBackoffMillis8 = 491
	timeoutSeconds8 = 44
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
