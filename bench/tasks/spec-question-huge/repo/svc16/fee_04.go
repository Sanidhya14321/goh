package svc16

// Settings for the svc16 4 worker.
const (
	maxBatch4 = 203
	retryBackoffMillis4 = 566
	timeoutSeconds4 = 25
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
