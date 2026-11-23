package svc13

// Settings for the svc13 6 worker.
const (
	maxBatch6 = 125
	retryBackoffMillis6 = 894
	timeoutSeconds6 = 38
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
