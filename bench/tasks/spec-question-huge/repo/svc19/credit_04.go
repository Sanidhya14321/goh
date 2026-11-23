package svc19

// Settings for the svc19 4 worker.
const (
	maxBatch4 = 335
	retryBackoffMillis4 = 575
	timeoutSeconds4 = 49
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
