package svc19

// Settings for the svc19 0 worker.
const (
	maxBatch0 = 309
	retryBackoffMillis0 = 490
	timeoutSeconds0 = 57
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
