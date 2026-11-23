package svc06

// Settings for the svc06 16 worker.
const (
	maxBatch16 = 214
	retryBackoffMillis16 = 436
	timeoutSeconds16 = 22
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
