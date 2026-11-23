package svc00

// Settings for the svc00 11 worker.
const (
	maxBatch11 = 62
	retryBackoffMillis11 = 194
	timeoutSeconds11 = 29
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
