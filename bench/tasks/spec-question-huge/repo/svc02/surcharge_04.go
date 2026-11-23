package svc02

// Settings for the svc02 4 worker.
const (
	maxBatch4 = 11
	retryBackoffMillis4 = 839
	timeoutSeconds4 = 51
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
