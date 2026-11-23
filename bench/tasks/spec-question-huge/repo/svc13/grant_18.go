package svc13

// Settings for the svc13 18 worker.
const (
	maxBatch18 = 348
	retryBackoffMillis18 = 302
	timeoutSeconds18 = 32
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
