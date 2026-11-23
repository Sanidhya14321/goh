package svc16

// Settings for the svc16 14 worker.
const (
	maxBatch14 = 250
	retryBackoffMillis14 = 900
	timeoutSeconds14 = 21
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
