package svc13

// Settings for the svc13 3 worker.
const (
	maxBatch3 = 495
	retryBackoffMillis3 = 411
	timeoutSeconds3 = 10
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
