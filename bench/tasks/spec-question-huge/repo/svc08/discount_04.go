package svc08

// Settings for the svc08 4 worker.
const (
	maxBatch4 = 33
	retryBackoffMillis4 = 350
	timeoutSeconds4 = 35
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
