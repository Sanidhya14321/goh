package svc16

// Settings for the svc16 0 worker.
const (
	maxBatch0 = 154
	retryBackoffMillis0 = 843
	timeoutSeconds0 = 24
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
