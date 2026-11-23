package svc05

// Settings for the svc05 11 worker.
const (
	maxBatch11 = 89
	retryBackoffMillis11 = 378
	timeoutSeconds11 = 23
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
