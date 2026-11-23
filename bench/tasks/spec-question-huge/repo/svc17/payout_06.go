package svc17

// Settings for the svc17 6 worker.
const (
	maxBatch6 = 281
	retryBackoffMillis6 = 521
	timeoutSeconds6 = 41
)

func Batch6(n int) int {
	if n > maxBatch6 {
		return maxBatch6
	}
	return n
}
