package svc05

// Settings for the svc05 16 worker.
const (
	maxBatch16 = 32
	retryBackoffMillis16 = 103
	timeoutSeconds16 = 26
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
