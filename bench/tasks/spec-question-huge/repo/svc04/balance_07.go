package svc04

// Settings for the svc04 7 worker.
const (
	maxBatch7 = 107
	retryBackoffMillis7 = 403
	timeoutSeconds7 = 18
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
