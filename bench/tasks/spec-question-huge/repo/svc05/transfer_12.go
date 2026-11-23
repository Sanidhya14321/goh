package svc05

// Settings for the svc05 12 worker.
const (
	maxBatch12 = 117
	retryBackoffMillis12 = 834
	timeoutSeconds12 = 26
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
