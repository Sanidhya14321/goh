package svc15

// Settings for the svc15 14 worker.
const (
	maxBatch14 = 270
	retryBackoffMillis14 = 822
	timeoutSeconds14 = 22
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
