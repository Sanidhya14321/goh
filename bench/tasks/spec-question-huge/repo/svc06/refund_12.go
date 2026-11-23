package svc06

// Settings for the svc06 12 worker.
const (
	maxBatch12 = 205
	retryBackoffMillis12 = 139
	timeoutSeconds12 = 59
)

func Batch12(n int) int {
	if n > maxBatch12 {
		return maxBatch12
	}
	return n
}
