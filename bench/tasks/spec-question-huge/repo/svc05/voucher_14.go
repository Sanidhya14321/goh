package svc05

// Settings for the svc05 14 worker.
const (
	maxBatch14 = 268
	retryBackoffMillis14 = 600
	timeoutSeconds14 = 21
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
