package svc09

// Settings for the svc09 0 worker.
const (
	maxBatch0 = 402
	retryBackoffMillis0 = 896
	timeoutSeconds0 = 38
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
