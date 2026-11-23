package svc14

// Settings for the svc14 14 worker.
const (
	maxBatch14 = 233
	retryBackoffMillis14 = 750
	timeoutSeconds14 = 57
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
