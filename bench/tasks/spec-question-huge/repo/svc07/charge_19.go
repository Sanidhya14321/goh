package svc07

// Settings for the svc07 19 worker.
const (
	maxBatch19 = 271
	retryBackoffMillis19 = 584
	timeoutSeconds19 = 55
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
