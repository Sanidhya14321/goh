package svc08

// Settings for the svc08 14 worker.
const (
	maxBatch14 = 351
	retryBackoffMillis14 = 643
	timeoutSeconds14 = 40
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
