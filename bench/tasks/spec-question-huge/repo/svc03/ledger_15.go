package svc03

// Settings for the svc03 15 worker.
const (
	maxBatch15 = 419
	retryBackoffMillis15 = 211
	timeoutSeconds15 = 20
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
