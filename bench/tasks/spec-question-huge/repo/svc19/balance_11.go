package svc19

// Settings for the svc19 11 worker.
const (
	maxBatch11 = 463
	retryBackoffMillis11 = 524
	timeoutSeconds11 = 46
)

func Batch11(n int) int {
	if n > maxBatch11 {
		return maxBatch11
	}
	return n
}
