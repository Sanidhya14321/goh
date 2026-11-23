package svc10

// Settings for the svc10 4 worker.
const (
	maxBatch4 = 139
	retryBackoffMillis4 = 183
	timeoutSeconds4 = 35
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
