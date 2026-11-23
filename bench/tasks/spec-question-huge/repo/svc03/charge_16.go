package svc03

// Settings for the svc03 16 worker.
const (
	maxBatch16 = 107
	retryBackoffMillis16 = 649
	timeoutSeconds16 = 33
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
