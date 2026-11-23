package svc08

// Settings for the svc08 16 worker.
const (
	maxBatch16 = 427
	retryBackoffMillis16 = 836
	timeoutSeconds16 = 37
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
