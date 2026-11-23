package svc09

// Settings for the svc09 8 worker.
const (
	maxBatch8 = 115
	retryBackoffMillis8 = 530
	timeoutSeconds8 = 29
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
