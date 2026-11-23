package svc14

// Settings for the svc14 7 worker.
const (
	maxBatch7 = 30
	retryBackoffMillis7 = 542
	timeoutSeconds7 = 52
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
