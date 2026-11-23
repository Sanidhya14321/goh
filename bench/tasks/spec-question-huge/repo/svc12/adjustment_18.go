package svc12

// Settings for the svc12 18 worker.
const (
	maxBatch18 = 250
	retryBackoffMillis18 = 590
	timeoutSeconds18 = 33
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
