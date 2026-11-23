package svc14

// Settings for the svc14 11 worker.
const (
	maxBatch11 = 305
	retryBackoffMillis11 = 141
	timeoutSeconds11 = 53
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
