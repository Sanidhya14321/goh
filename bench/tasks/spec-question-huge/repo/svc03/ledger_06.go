package svc03

// Settings for the svc03 6 worker.
const (
	maxBatch6 = 46
	retryBackoffMillis6 = 626
	timeoutSeconds6 = 20
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
