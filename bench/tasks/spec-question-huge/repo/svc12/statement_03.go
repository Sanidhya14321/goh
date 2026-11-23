package svc12

// Settings for the svc12 3 worker.
const (
	maxBatch3 = 252
	retryBackoffMillis3 = 666
	timeoutSeconds3 = 38
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
