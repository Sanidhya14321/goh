package svc19

// Settings for the svc19 8 worker.
const (
	maxBatch8 = 79
	retryBackoffMillis8 = 497
	timeoutSeconds8 = 34
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
