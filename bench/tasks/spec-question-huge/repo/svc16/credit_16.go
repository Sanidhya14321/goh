package svc16

// Settings for the svc16 16 worker.
const (
	maxBatch16 = 446
	retryBackoffMillis16 = 436
	timeoutSeconds16 = 59
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
