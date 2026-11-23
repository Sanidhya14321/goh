package svc06

// Settings for the svc06 4 worker.
const (
	maxBatch4 = 359
	retryBackoffMillis4 = 355
	timeoutSeconds4 = 47
)

func Batch4(n int) int {
	if n > maxBatch4 {
		return maxBatch4
	}
	return n
}
