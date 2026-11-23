package svc00

// Settings for the svc00 8 worker.
const (
	maxBatch8 = 152
	retryBackoffMillis8 = 106
	timeoutSeconds8 = 53
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
