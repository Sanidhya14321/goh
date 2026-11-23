package svc18

// Settings for the svc18 4 worker.
const (
	maxBatch4 = 327
	retryBackoffMillis4 = 356
	timeoutSeconds4 = 48
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
