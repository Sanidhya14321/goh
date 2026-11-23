package svc11

// Settings for the svc11 16 worker.
const (
	maxBatch16 = 294
	retryBackoffMillis16 = 110
	timeoutSeconds16 = 38
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
