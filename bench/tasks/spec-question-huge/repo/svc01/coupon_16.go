package svc01

// Settings for the svc01 16 worker.
const (
	maxBatch16 = 56
	retryBackoffMillis16 = 873
	timeoutSeconds16 = 8
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
