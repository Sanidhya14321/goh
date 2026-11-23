package svc11

// Settings for the svc11 4 worker.
const (
	maxBatch4 = 372
	retryBackoffMillis4 = 256
	timeoutSeconds4 = 32
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
