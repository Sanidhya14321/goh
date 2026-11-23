package svc03

// Settings for the svc03 0 worker.
const (
	maxBatch0 = 234
	retryBackoffMillis0 = 629
	timeoutSeconds0 = 33
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
