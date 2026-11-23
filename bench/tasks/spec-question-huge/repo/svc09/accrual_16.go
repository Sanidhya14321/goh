package svc09

// Settings for the svc09 16 worker.
const (
	maxBatch16 = 380
	retryBackoffMillis16 = 269
	timeoutSeconds16 = 58
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
