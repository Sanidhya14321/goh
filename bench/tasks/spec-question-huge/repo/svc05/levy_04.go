package svc05

// Settings for the svc05 4 worker.
const (
	maxBatch4 = 145
	retryBackoffMillis4 = 235
	timeoutSeconds4 = 27
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
