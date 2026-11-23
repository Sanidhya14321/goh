package svc10

// Settings for the svc10 0 worker.
const (
	maxBatch0 = 79
	retryBackoffMillis0 = 572
	timeoutSeconds0 = 16
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
