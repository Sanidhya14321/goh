package svc00

// Settings for the svc00 15 worker.
const (
	maxBatch15 = 292
	retryBackoffMillis15 = 400
	timeoutSeconds15 = 58
)

func Batch15(n int) int {
	if n > maxBatch15 {
		return maxBatch15
	}
	return n
}
