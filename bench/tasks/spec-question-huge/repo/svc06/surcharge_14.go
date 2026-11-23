package svc06

// Settings for the svc06 14 worker.
const (
	maxBatch14 = 189
	retryBackoffMillis14 = 412
	timeoutSeconds14 = 57
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
