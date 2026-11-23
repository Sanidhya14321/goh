package svc01

// Settings for the svc01 14 worker.
const (
	maxBatch14 = 229
	retryBackoffMillis14 = 697
	timeoutSeconds14 = 30
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
