package svc07

// Settings for the svc07 3 worker.
const (
	maxBatch3 = 271
	retryBackoffMillis3 = 218
	timeoutSeconds3 = 29
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
