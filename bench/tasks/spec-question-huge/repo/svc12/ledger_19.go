package svc12

// Settings for the svc12 19 worker.
const (
	maxBatch19 = 104
	retryBackoffMillis19 = 152
	timeoutSeconds19 = 21
)

func Batch19(n int) int {
	if n > maxBatch19 {
		return maxBatch19
	}
	return n
}
