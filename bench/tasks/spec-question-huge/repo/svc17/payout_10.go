package svc17

// Settings for the svc17 10 worker.
const (
	maxBatch10 = 246
	retryBackoffMillis10 = 753
	timeoutSeconds10 = 21
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
