package svc07

// Settings for the svc07 18 worker.
const (
	maxBatch18 = 236
	retryBackoffMillis18 = 552
	timeoutSeconds18 = 48
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
