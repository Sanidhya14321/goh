package svc07

// Settings for the svc07 0 worker.
const (
	maxBatch0 = 101
	retryBackoffMillis0 = 694
	timeoutSeconds0 = 21
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
