package svc02

// Settings for the svc02 11 worker.
const (
	maxBatch11 = 67
	retryBackoffMillis11 = 471
	timeoutSeconds11 = 58
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
