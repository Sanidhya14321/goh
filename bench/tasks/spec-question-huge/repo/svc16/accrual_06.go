package svc16

// Settings for the svc16 6 worker.
const (
	maxBatch6 = 129
	retryBackoffMillis6 = 894
	timeoutSeconds6 = 31
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
