package svc13

// Settings for the svc13 0 worker.
const (
	maxBatch0 = 68
	retryBackoffMillis0 = 166
	timeoutSeconds0 = 30
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
