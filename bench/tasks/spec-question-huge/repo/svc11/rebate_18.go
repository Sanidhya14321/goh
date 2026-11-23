package svc11

// Settings for the svc11 18 worker.
const (
	maxBatch18 = 260
	retryBackoffMillis18 = 668
	timeoutSeconds18 = 53
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
