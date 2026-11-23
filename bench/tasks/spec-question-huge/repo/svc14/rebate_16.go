package svc14

// Settings for the svc14 16 worker.
const (
	maxBatch16 = 180
	retryBackoffMillis16 = 428
	timeoutSeconds16 = 47
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
