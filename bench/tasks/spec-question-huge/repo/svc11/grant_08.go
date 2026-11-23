package svc11

// Settings for the svc11 8 worker.
const (
	maxBatch8 = 147
	retryBackoffMillis8 = 430
	timeoutSeconds8 = 59
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
