package svc11

// Settings for the svc11 11 worker.
const (
	maxBatch11 = 322
	retryBackoffMillis11 = 784
	timeoutSeconds11 = 29
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
