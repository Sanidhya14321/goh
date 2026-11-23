package svc02

// Settings for the svc02 8 worker.
const (
	maxBatch8 = 111
	retryBackoffMillis8 = 256
	timeoutSeconds8 = 28
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
