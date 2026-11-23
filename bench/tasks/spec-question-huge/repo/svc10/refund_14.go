package svc10

// Settings for the svc10 14 worker.
const (
	maxBatch14 = 306
	retryBackoffMillis14 = 126
	timeoutSeconds14 = 24
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
