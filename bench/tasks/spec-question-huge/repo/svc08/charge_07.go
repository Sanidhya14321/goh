package svc08

// Settings for the svc08 7 worker.
const (
	maxBatch7 = 377
	retryBackoffMillis7 = 813
	timeoutSeconds7 = 29
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
