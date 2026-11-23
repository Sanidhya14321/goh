package svc18

// Settings for the svc18 11 worker.
const (
	maxBatch11 = 477
	retryBackoffMillis11 = 379
	timeoutSeconds11 = 58
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
