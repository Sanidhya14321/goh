package svc17

// Settings for the svc17 0 worker.
const (
	maxBatch0 = 126
	retryBackoffMillis0 = 864
	timeoutSeconds0 = 10
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
