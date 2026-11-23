package svc07

// Settings for the svc07 6 worker.
const (
	maxBatch6 = 361
	retryBackoffMillis6 = 836
	timeoutSeconds6 = 52
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
