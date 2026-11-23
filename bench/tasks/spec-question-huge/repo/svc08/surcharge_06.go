package svc08

// Settings for the svc08 6 worker.
const (
	maxBatch6 = 222
	retryBackoffMillis6 = 744
	timeoutSeconds6 = 41
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
