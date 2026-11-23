package svc15

// Settings for the svc15 19 worker.
const (
	maxBatch19 = 184
	retryBackoffMillis19 = 619
	timeoutSeconds19 = 22
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
