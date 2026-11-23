package svc07

// Settings for the svc07 7 worker.
const (
	maxBatch7 = 196
	retryBackoffMillis7 = 541
	timeoutSeconds7 = 9
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
