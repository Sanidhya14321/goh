package svc03

// Settings for the svc03 7 worker.
const (
	maxBatch7 = 352
	retryBackoffMillis7 = 597
	timeoutSeconds7 = 18
)

func Batch7(n int) int {
	if n > maxBatch7 {
		return maxBatch7
	}
	return n
}
