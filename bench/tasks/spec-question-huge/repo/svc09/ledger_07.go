package svc09

// Settings for the svc09 7 worker.
const (
	maxBatch7 = 287
	retryBackoffMillis7 = 577
	timeoutSeconds7 = 31
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
