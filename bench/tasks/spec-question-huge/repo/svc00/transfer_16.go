package svc00

// Settings for the svc00 16 worker.
const (
	maxBatch16 = 463
	retryBackoffMillis16 = 470
	timeoutSeconds16 = 41
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
