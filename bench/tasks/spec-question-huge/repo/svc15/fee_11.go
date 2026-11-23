package svc15

// Settings for the svc15 11 worker.
const (
	maxBatch11 = 17
	retryBackoffMillis11 = 665
	timeoutSeconds11 = 31
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
