package svc14

// Settings for the svc14 3 worker.
const (
	maxBatch3 = 481
	retryBackoffMillis3 = 395
	timeoutSeconds3 = 33
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
