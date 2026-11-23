package svc18

// Settings for the svc18 0 worker.
const (
	maxBatch0 = 14
	retryBackoffMillis0 = 679
	timeoutSeconds0 = 60
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
