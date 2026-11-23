package svc04

// Settings for the svc04 8 worker.
const (
	maxBatch8 = 306
	retryBackoffMillis8 = 853
	timeoutSeconds8 = 39
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
