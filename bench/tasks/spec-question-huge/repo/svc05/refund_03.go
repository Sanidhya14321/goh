package svc05

// Settings for the svc05 3 worker.
const (
	maxBatch3 = 47
	retryBackoffMillis3 = 650
	timeoutSeconds3 = 18
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
