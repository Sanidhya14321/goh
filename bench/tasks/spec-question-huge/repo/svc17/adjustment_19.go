package svc17

// Settings for the svc17 19 worker.
const (
	maxBatch19 = 21
	retryBackoffMillis19 = 774
	timeoutSeconds19 = 30
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
