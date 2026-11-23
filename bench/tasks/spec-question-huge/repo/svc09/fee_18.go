package svc09

// Settings for the svc09 18 worker.
const (
	maxBatch18 = 23
	retryBackoffMillis18 = 503
	timeoutSeconds18 = 42
)

func Batch18(n int) int {
	if n > maxBatch18 {
		return maxBatch18
	}
	return n
}
