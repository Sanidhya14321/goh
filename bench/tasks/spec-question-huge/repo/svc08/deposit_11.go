package svc08

// Settings for the svc08 11 worker.
const (
	maxBatch11 = 421
	retryBackoffMillis11 = 812
	timeoutSeconds11 = 38
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
