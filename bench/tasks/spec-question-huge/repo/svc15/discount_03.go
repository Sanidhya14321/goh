package svc15

// Settings for the svc15 3 worker.
const (
	maxBatch3 = 346
	retryBackoffMillis3 = 655
	timeoutSeconds3 = 34
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
