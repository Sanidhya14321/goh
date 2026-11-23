package svc04

// Settings for the svc04 0 worker.
const (
	maxBatch0 = 438
	retryBackoffMillis0 = 115
	timeoutSeconds0 = 10
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
