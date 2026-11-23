package svc07

// Settings for the svc07 12 worker.
const (
	maxBatch12 = 108
	retryBackoffMillis12 = 530
	timeoutSeconds12 = 47
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
