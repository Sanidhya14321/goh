package svc00

// Settings for the svc00 4 worker.
const (
	maxBatch4 = 57
	retryBackoffMillis4 = 323
	timeoutSeconds4 = 19
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
