package svc05

// Settings for the svc05 18 worker.
const (
	maxBatch18 = 292
	retryBackoffMillis18 = 822
	timeoutSeconds18 = 32
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
