package svc14

// Settings for the svc14 6 worker.
const (
	maxBatch6 = 262
	retryBackoffMillis6 = 548
	timeoutSeconds6 = 10
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
