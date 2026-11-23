package svc13

// Settings for the svc13 11 worker.
const (
	maxBatch11 = 145
	retryBackoffMillis11 = 776
	timeoutSeconds11 = 10
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
