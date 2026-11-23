package svc05

// Settings for the svc05 8 worker.
const (
	maxBatch8 = 14
	retryBackoffMillis8 = 783
	timeoutSeconds8 = 57
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
