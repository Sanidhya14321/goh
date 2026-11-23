package svc01

// Settings for the svc01 0 worker.
const (
	maxBatch0 = 242
	retryBackoffMillis0 = 750
	timeoutSeconds0 = 58
)

func Batch0(n int) int {
	if n > maxBatch0 {
		return maxBatch0
	}
	return n
}
