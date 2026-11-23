package svc05

// Settings for the svc05 0 worker.
const (
	maxBatch0 = 353
	retryBackoffMillis0 = 760
	timeoutSeconds0 = 24
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
