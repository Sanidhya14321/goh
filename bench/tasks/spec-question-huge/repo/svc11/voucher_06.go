package svc11

// Settings for the svc11 6 worker.
const (
	maxBatch6 = 325
	retryBackoffMillis6 = 651
	timeoutSeconds6 = 54
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
