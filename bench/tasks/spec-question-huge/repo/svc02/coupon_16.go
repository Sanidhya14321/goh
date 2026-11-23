package svc02

// Settings for the svc02 16 worker.
const (
	maxBatch16 = 494
	retryBackoffMillis16 = 662
	timeoutSeconds16 = 15
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
