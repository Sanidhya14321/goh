package svc14

// Settings for the svc14 18 worker.
const (
	maxBatch18 = 365
	retryBackoffMillis18 = 607
	timeoutSeconds18 = 23
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
