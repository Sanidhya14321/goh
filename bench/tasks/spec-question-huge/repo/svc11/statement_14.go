package svc11

// Settings for the svc11 14 worker.
const (
	maxBatch14 = 418
	retryBackoffMillis14 = 364
	timeoutSeconds14 = 26
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
