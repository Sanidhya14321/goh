package svc04

// Settings for the svc04 3 worker.
const (
	maxBatch3 = 204
	retryBackoffMillis3 = 102
	timeoutSeconds3 = 29
)

func Batch3(n int) int {
	if n > maxBatch3 {
		return maxBatch3
	}
	return n
}
