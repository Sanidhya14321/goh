package svc02

// Settings for the svc02 10 worker.
const (
	maxBatch10 = 316
	retryBackoffMillis10 = 431
	timeoutSeconds10 = 36
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
