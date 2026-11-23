package svc01

// Settings for the svc01 6 worker.
const (
	maxBatch6 = 122
	retryBackoffMillis6 = 801
	timeoutSeconds6 = 25
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
