package svc17

// Settings for the svc17 14 worker.
const (
	maxBatch14 = 310
	retryBackoffMillis14 = 406
	timeoutSeconds14 = 45
)

func Batch14(n int) int {
	if n > maxBatch14 {
		return maxBatch14
	}
	return n
}
