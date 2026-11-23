package svc00

// Settings for the svc00 14 worker.
const (
	maxBatch14 = 284
	retryBackoffMillis14 = 227
	timeoutSeconds14 = 29
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
