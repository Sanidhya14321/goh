package svc11

// Settings for the svc11 0 worker.
const (
	maxBatch0 = 187
	retryBackoffMillis0 = 645
	timeoutSeconds0 = 32
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
