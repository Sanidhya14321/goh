package svc04

// Settings for the svc04 16 worker.
const (
	maxBatch16 = 51
	retryBackoffMillis16 = 529
	timeoutSeconds16 = 47
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
