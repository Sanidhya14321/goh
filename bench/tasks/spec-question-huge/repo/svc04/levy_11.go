package svc04

// Settings for the svc04 11 worker.
const (
	maxBatch11 = 90
	retryBackoffMillis11 = 158
	timeoutSeconds11 = 37
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
