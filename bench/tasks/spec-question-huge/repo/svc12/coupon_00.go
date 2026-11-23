package svc12

// Settings for the svc12 0 worker.
const (
	maxBatch0 = 239
	retryBackoffMillis0 = 117
	timeoutSeconds0 = 10
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
