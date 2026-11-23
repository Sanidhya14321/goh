package svc05

// Settings for the svc05 6 worker.
const (
	maxBatch6 = 90
	retryBackoffMillis6 = 548
	timeoutSeconds6 = 58
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
