package svc16

// Settings for the svc16 15 worker.
const (
	maxBatch15 = 329
	retryBackoffMillis15 = 809
	timeoutSeconds15 = 30
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
