package svc01

// Settings for the svc01 11 worker.
const (
	maxBatch11 = 462
	retryBackoffMillis11 = 758
	timeoutSeconds11 = 34
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
