package svc01

// Settings for the svc01 7 worker.
const (
	maxBatch7 = 127
	retryBackoffMillis7 = 132
	timeoutSeconds7 = 56
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
