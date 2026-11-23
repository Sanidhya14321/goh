package svc02

// Settings for the svc02 0 worker.
const (
	maxBatch0 = 293
	retryBackoffMillis0 = 111
	timeoutSeconds0 = 48
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
