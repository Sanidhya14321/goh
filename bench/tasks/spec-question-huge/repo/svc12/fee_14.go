package svc12

// Settings for the svc12 14 worker.
const (
	maxBatch14 = 74
	retryBackoffMillis14 = 380
	timeoutSeconds14 = 7
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
