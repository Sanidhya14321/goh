package svc07

// Settings for the svc07 10 worker.
const (
	maxBatch10 = 351
	retryBackoffMillis10 = 518
	timeoutSeconds10 = 25
)

func Batch10(n int) int {
	if n > maxBatch10 {
		return maxBatch10
	}
	return n
}
