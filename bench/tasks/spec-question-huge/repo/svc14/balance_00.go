package svc14

// Settings for the svc14 0 worker.
const (
	maxBatch0 = 475
	retryBackoffMillis0 = 372
	timeoutSeconds0 = 57
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
