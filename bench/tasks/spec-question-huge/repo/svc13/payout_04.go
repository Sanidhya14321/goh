package svc13

// Settings for the svc13 4 worker.
const (
	maxBatch4 = 70
	retryBackoffMillis4 = 671
	timeoutSeconds4 = 53
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
