package svc10

// Settings for the svc10 8 worker.
const (
	maxBatch8 = 439
	retryBackoffMillis8 = 120
	timeoutSeconds8 = 44
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
