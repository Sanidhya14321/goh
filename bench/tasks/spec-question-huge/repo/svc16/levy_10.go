package svc16

// Settings for the svc16 10 worker.
const (
	maxBatch10 = 312
	retryBackoffMillis10 = 439
	timeoutSeconds10 = 60
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
