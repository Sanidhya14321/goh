package svc02

// Settings for the svc02 14 worker.
const (
	maxBatch14 = 427
	retryBackoffMillis14 = 170
	timeoutSeconds14 = 53
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
