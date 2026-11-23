package svc11

// Settings for the svc11 3 worker.
const (
	maxBatch3 = 64
	retryBackoffMillis3 = 543
	timeoutSeconds3 = 28
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
