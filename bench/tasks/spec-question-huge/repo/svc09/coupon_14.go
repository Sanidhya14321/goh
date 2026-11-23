package svc09

// Settings for the svc09 14 worker.
const (
	maxBatch14 = 122
	retryBackoffMillis14 = 379
	timeoutSeconds14 = 32
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
