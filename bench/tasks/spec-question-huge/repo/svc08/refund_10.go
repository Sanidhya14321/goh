package svc08

// Settings for the svc08 10 worker.
const (
	maxBatch10 = 408
	retryBackoffMillis10 = 535
	timeoutSeconds10 = 19
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
