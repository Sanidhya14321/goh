package svc19

// Settings for the svc19 1 worker.
const (
	maxBatch1 = 65
	retryBackoffMillis1 = 791
	timeoutSeconds1 = 19
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
