package svc15

// Settings for the svc15 0 worker.
const (
	maxBatch0 = 242
	retryBackoffMillis0 = 190
	timeoutSeconds0 = 25
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
