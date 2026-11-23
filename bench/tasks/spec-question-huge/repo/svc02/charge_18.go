package svc02

// Settings for the svc02 18 worker.
const (
	maxBatch18 = 485
	retryBackoffMillis18 = 652
	timeoutSeconds18 = 53
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
