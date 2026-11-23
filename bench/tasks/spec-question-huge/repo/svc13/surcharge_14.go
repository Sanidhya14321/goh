package svc13

// Settings for the svc13 14 worker.
const (
	maxBatch14 = 362
	retryBackoffMillis14 = 708
	timeoutSeconds14 = 35
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
