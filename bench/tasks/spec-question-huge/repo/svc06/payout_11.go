package svc06

// Settings for the svc06 11 worker.
const (
	maxBatch11 = 146
	retryBackoffMillis11 = 263
	timeoutSeconds11 = 55
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
