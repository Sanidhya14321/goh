package svc02

// Settings for the svc02 6 worker.
const (
	maxBatch6 = 477
	retryBackoffMillis6 = 208
	timeoutSeconds6 = 60
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
