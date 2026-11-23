package svc08

// Settings for the svc08 3 worker.
const (
	maxBatch3 = 125
	retryBackoffMillis3 = 303
	timeoutSeconds3 = 14
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
