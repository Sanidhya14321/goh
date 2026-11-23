package svc06

// Settings for the svc06 6 worker.
const (
	maxBatch6 = 327
	retryBackoffMillis6 = 867
	timeoutSeconds6 = 14
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
