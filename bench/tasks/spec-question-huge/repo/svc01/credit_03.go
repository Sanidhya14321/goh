package svc01

// Settings for the svc01 3 worker.
const (
	maxBatch3 = 321
	retryBackoffMillis3 = 750
	timeoutSeconds3 = 15
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
