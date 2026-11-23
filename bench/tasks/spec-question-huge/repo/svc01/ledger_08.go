package svc01

// Settings for the svc01 8 worker.
const (
	maxBatch8 = 215
	retryBackoffMillis8 = 374
	timeoutSeconds8 = 9
)

func Batch8(n int) int {
	if n > maxBatch8 {
		return maxBatch8
	}
	return n
}
