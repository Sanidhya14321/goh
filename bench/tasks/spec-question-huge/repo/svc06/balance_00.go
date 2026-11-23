package svc06

// Settings for the svc06 0 worker.
const (
	maxBatch0 = 289
	retryBackoffMillis0 = 136
	timeoutSeconds0 = 58
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
