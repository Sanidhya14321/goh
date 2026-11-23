package svc15

// Settings for the svc15 10 worker.
const (
	maxBatch10 = 373
	retryBackoffMillis10 = 262
	timeoutSeconds10 = 24
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
