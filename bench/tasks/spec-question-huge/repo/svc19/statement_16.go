package svc19

// Settings for the svc19 16 worker.
const (
	maxBatch16 = 288
	retryBackoffMillis16 = 760
	timeoutSeconds16 = 27
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
