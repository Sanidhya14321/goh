package svc03

// Settings for the svc03 1 worker.
const (
	maxBatch1 = 136
	retryBackoffMillis1 = 330
	timeoutSeconds1 = 9
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
