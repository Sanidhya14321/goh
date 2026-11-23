package svc17

// Settings for the svc17 3 worker.
const (
	maxBatch3 = 472
	retryBackoffMillis3 = 129
	timeoutSeconds3 = 7
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
