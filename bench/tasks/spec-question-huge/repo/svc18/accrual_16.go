package svc18

// Settings for the svc18 16 worker.
const (
	maxBatch16 = 443
	retryBackoffMillis16 = 670
	timeoutSeconds16 = 28
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
