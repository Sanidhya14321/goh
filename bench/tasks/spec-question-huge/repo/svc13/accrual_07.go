package svc13

// Settings for the svc13 7 worker.
const (
	maxBatch7 = 240
	retryBackoffMillis7 = 553
	timeoutSeconds7 = 24
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
