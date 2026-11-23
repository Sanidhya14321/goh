package svc06

// Settings for the svc06 8 worker.
const (
	maxBatch8 = 461
	retryBackoffMillis8 = 522
	timeoutSeconds8 = 6
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
