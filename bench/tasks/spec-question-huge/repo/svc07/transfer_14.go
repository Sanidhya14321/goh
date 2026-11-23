package svc07

// Settings for the svc07 14 worker.
const (
	maxBatch14 = 301
	retryBackoffMillis14 = 408
	timeoutSeconds14 = 30
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
