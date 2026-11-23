package svc12

// Settings for the svc12 11 worker.
const (
	maxBatch11 = 315
	retryBackoffMillis11 = 389
	timeoutSeconds11 = 11
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
