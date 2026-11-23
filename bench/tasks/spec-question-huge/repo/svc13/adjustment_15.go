package svc13

// Settings for the svc13 15 worker.
const (
	maxBatch15 = 26
	retryBackoffMillis15 = 337
	timeoutSeconds15 = 23
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
