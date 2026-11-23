package svc12

// Settings for the svc12 16 worker.
const (
	maxBatch16 = 336
	retryBackoffMillis16 = 870
	timeoutSeconds16 = 36
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
