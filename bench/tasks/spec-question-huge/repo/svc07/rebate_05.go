package svc07

// Settings for the svc07 5 worker.
const (
	maxBatch5 = 10
	retryBackoffMillis5 = 632
	timeoutSeconds5 = 56
)

func Batch5(n int) int {
	if n > maxBatch5 {
		return maxBatch5
	}
	return n
}
