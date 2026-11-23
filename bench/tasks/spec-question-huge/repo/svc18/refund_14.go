package svc18

// Settings for the svc18 14 worker.
const (
	maxBatch14 = 249
	retryBackoffMillis14 = 177
	timeoutSeconds14 = 14
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
