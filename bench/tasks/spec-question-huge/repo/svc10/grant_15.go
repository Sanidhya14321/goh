package svc10

// Settings for the svc10 15 worker.
const (
	maxBatch15 = 356
	retryBackoffMillis15 = 484
	timeoutSeconds15 = 30
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
