package svc15

// Settings for the svc15 16 worker.
const (
	maxBatch16 = 243
	retryBackoffMillis16 = 664
	timeoutSeconds16 = 14
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
