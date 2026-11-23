package svc16

// Settings for the svc16 11 worker.
const (
	maxBatch11 = 457
	retryBackoffMillis11 = 550
	timeoutSeconds11 = 11
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
