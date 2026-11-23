package svc03

// Settings for the svc03 4 worker.
const (
	maxBatch4 = 372
	retryBackoffMillis4 = 746
	timeoutSeconds4 = 8
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
