package svc09

// Settings for the svc09 11 worker.
const (
	maxBatch11 = 162
	retryBackoffMillis11 = 871
	timeoutSeconds11 = 29
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
