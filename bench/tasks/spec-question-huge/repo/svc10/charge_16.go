package svc10

// Settings for the svc10 16 worker.
const (
	maxBatch16 = 48
	retryBackoffMillis16 = 706
	timeoutSeconds16 = 49
)

func Batch16(n int) int {
	if n > maxBatch16 {
		return maxBatch16
	}
	return n
}
