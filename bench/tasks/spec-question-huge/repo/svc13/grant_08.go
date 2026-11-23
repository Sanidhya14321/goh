package svc13

// Settings for the svc13 8 worker.
const (
	maxBatch8 = 229
	retryBackoffMillis8 = 412
	timeoutSeconds8 = 41
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
