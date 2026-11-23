package svc08

// Settings for the svc08 0 worker.
const (
	maxBatch0 = 347
	retryBackoffMillis0 = 186
	timeoutSeconds0 = 23
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
