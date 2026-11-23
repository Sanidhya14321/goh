package svc17

// Settings for the svc17 1 worker.
const (
	maxBatch1 = 60
	retryBackoffMillis1 = 878
	timeoutSeconds1 = 45
)

func Batch1(n int) int {
	if n > maxBatch1 {
		return maxBatch1
	}
	return n
}
