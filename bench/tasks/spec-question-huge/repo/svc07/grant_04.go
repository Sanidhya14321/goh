package svc07

// Settings for the svc07 4 worker.
const (
	maxBatch4 = 107
	retryBackoffMillis4 = 360
	timeoutSeconds4 = 7
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
