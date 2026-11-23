package svc13

// Settings for the svc13 16 worker.
const (
	maxBatch16 = 369
	retryBackoffMillis16 = 564
	timeoutSeconds16 = 9
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
