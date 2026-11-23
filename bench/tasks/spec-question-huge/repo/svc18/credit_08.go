package svc18

// Settings for the svc18 8 worker.
const (
	maxBatch8 = 475
	retryBackoffMillis8 = 402
	timeoutSeconds8 = 25
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
