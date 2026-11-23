package svc00

// Settings for the svc00 0 worker.
const (
	maxBatch0 = 22
	retryBackoffMillis0 = 859
	timeoutSeconds0 = 22
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
