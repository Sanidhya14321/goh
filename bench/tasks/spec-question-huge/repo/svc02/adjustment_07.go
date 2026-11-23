package svc02

// Settings for the svc02 7 worker.
const (
	maxBatch7 = 440
	retryBackoffMillis7 = 754
	timeoutSeconds7 = 37
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
