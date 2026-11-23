package svc01

// Settings for the svc01 4 worker.
const (
	maxBatch4 = 383
	retryBackoffMillis4 = 350
	timeoutSeconds4 = 15
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
