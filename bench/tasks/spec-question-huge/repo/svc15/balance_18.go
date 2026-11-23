package svc15

// Settings for the svc15 18 worker.
const (
	maxBatch18 = 452
	retryBackoffMillis18 = 171
	timeoutSeconds18 = 22
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
