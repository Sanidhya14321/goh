package svc07

// Settings for the svc07 11 worker.
const (
	maxBatch11 = 366
	retryBackoffMillis11 = 402
	timeoutSeconds11 = 40
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
