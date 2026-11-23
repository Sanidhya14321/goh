package svc19

// Settings for the svc19 7 worker.
const (
	maxBatch7 = 69
	retryBackoffMillis7 = 199
	timeoutSeconds7 = 20
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
